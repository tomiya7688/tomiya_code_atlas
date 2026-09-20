"""Shared host for Parser Backend Contract v1 JSON helper executables."""

from __future__ import annotations

import json
import subprocess
import uuid
from pathlib import Path
from typing import Any

from Src.analyzers.ir import CodeEntity, DiagnosticIR, EntityKind, ModuleIR, Visibility
from Src.data.backend_assets import resolve_backend_asset
from Src.languages.backend import (
    PARSER_BACKEND_CONTRACT_VERSION,
    ParserBackendDescriptor,
    ParserBackendError,
    ParserBackendFailure,
    ParserBackendFailureKind,
)


def module_from_wire_ir(payload: dict[str, Any]) -> ModuleIR:
    """Convert a language-independent helper DTO into passive Common IR."""
    entities: list[CodeEntity] = []
    for item in payload.get("entities", ()):
        try:
            kind = EntityKind(str(item["kind"]))
        except (KeyError, ValueError, TypeError) as error:
            raise ValueError("Helper returned an unsupported entity kind.") from error
        try:
            visibility = Visibility(str(item.get("visibility", "unspecified")))
        except ValueError:
            visibility = Visibility.UNSPECIFIED
        entities.append(
            CodeEntity(
                kind=kind,
                name=str(item["name"]),
                line=int(item.get("line", 1)),
                end_line=int(item.get("end_line", item.get("line", 1))),
                parent=item.get("parent"),
                parameters=tuple(str(value) for value in item.get("parameters", ())),
                decorators=tuple(str(value) for value in item.get("decorators", ())),
                calls=tuple(str(value) for value in item.get("calls", ())),
                call_sequence=tuple(str(value) for value in item.get("call_sequence", ())),
                visibility=visibility,
                bases=tuple(str(value) for value in item.get("bases", ())),
                declaration_kind=item.get("declaration_kind"),
                type_parameters=tuple(str(value) for value in item.get("type_parameters", ())),
                type_constraints=tuple(str(value) for value in item.get("type_constraints", ())),
                resolved_calls=tuple(str(value) for value in item.get("resolved_calls", ())),
                symbol_id=item.get("symbol_id"),
                is_async=bool(item.get("is_async", False)),
            )
        )

    diagnostics = [
        DiagnosticIR(
            str(item.get("kind", "info")),
            str(item.get("message", "")),
            int(item["line"]) if item.get("line") is not None else None,
        )
        for item in payload.get("diagnostics", ())
    ]
    return ModuleIR(
        language=str(payload.get("language", "")),
        entities=entities,
        diagnostics=diagnostics,
        imports=tuple(str(value) for value in payload.get("imports", ())),
    )


class JsonHelperBackend:
    """Invoke one bundled JSON helper without exposing its implementation types."""

    descriptor: ParserBackendDescriptor
    helper_relative_path: str

    def __init__(
        self,
        *,
        app_root: Path | None = None,
        timeout_seconds: float = 30.0,
    ) -> None:
        self._app_root = app_root
        self._timeout_seconds = timeout_seconds

    def helper_command(self, helper: Path) -> list[str]:
        """Return the subprocess command for this helper asset."""
        return [str(helper)]

    def parse(self, source: str, path: str | None = None) -> ModuleIR:
        helper = resolve_backend_asset(self.helper_relative_path, self._app_root)
        if not helper.is_file():
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.FAILURE,
                    f"Parser backend helper not found: {helper}",
                    self.descriptor.backend_id,
                )
            )

        request_id = uuid.uuid4().hex
        request = {
            "contract_version": PARSER_BACKEND_CONTRACT_VERSION,
            "request_id": request_id,
            "operation": "parse",
            "language": self.descriptor.language,
            "source": source,
            "path": path,
        }

        try:
            completed = subprocess.run(
                self.helper_command(helper),
                input=json.dumps(request),
                text=True,
                encoding="utf-8",
                capture_output=True,
                timeout=self._timeout_seconds,
                check=False,
            )
        except subprocess.TimeoutExpired as error:
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.TIMEOUT,
                    f"Parser backend timed out after {self._timeout_seconds:g}s.",
                    self.descriptor.backend_id,
                    retryable=True,
                )
            ) from error
        except OSError as error:
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.FAILURE,
                    str(error),
                    self.descriptor.backend_id,
                )
            ) from error

        if completed.returncode != 0:
            message = completed.stderr.strip() or f"Helper exited with code {completed.returncode}."
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.FAILURE,
                    message,
                    self.descriptor.backend_id,
                )
            )

        try:
            response = json.loads(completed.stdout)
        except json.JSONDecodeError as error:
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.PROTOCOL_ERROR,
                    "Parser helper returned invalid JSON.",
                    self.descriptor.backend_id,
                )
            ) from error

        if (
            response.get("contract_version") != PARSER_BACKEND_CONTRACT_VERSION
            or response.get("request_id") != request_id
        ):
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.PROTOCOL_ERROR,
                    "Parser helper response contract or request id does not match.",
                    self.descriptor.backend_id,
                )
            )

        if not response.get("ok"):
            payload = response.get("error") or {}
            try:
                kind = ParserBackendFailureKind(str(payload.get("kind", "failure")))
            except ValueError:
                kind = ParserBackendFailureKind.PROTOCOL_ERROR
            raise ParserBackendError(
                ParserBackendFailure(
                    kind,
                    str(payload.get("message", "Parser backend failed.")),
                    self.descriptor.backend_id,
                    bool(payload.get("retryable", False)),
                )
            )

        ir_payload = response.get("ir")
        if not isinstance(ir_payload, dict):
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.PROTOCOL_ERROR,
                    "Parser helper response is missing Common IR payload.",
                    self.descriptor.backend_id,
                )
            )

        try:
            module = module_from_wire_ir(ir_payload)
        except (TypeError, ValueError, KeyError) as error:
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.PROTOCOL_ERROR,
                    f"Parser helper Common IR payload is invalid: {error}",
                    self.descriptor.backend_id,
                )
            ) from error
        if module.language != self.descriptor.language:
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.PROTOCOL_ERROR,
                    "Parser helper returned Common IR for the wrong language.",
                    self.descriptor.backend_id,
                )
            )
        return module
