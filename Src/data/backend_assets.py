"""Runtime discovery for bundled parser backend assets."""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any


BACKEND_ASSET_DIRECTORY = "backends"
BACKEND_MANIFEST_NAME = "manifest.json"
BACKEND_MANIFEST_VERSION = "1"


def runtime_root() -> Path:
    """Return the application root for source and frozen onedir execution."""
    if getattr(sys, "frozen", False):
        return Path(sys.executable).resolve().parent
    return Path(__file__).resolve().parents[2]


def backend_asset_root(app_root: Path | None = None) -> Path:
    """Return the directory reserved for bundled helper/native parser assets."""
    root = app_root or runtime_root()
    packaged_root = root / BACKEND_ASSET_DIRECTORY
    if (packaged_root / BACKEND_MANIFEST_NAME).is_file():
        return packaged_root

    source_manifest = root / "parser-backends" / BACKEND_MANIFEST_NAME
    if source_manifest.is_file():
        try:
            payload = json.loads(source_manifest.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            return packaged_root
        build_output_root = payload.get("build_output_root")
        if isinstance(build_output_root, str):
            relative_path = Path(build_output_root)
            if not relative_path.is_absolute() and ".." not in relative_path.parts:
                candidate = (root / relative_path).resolve()
                resolved_root = root.resolve()
                if resolved_root in candidate.parents and (
                    candidate / BACKEND_MANIFEST_NAME
                ).is_file():
                    return candidate
    return packaged_root


def load_backend_manifest(app_root: Path | None = None) -> dict[str, Any]:
    """Read and minimally validate the bundled backend manifest."""
    app_root = app_root or runtime_root()
    path = backend_asset_root(app_root) / BACKEND_MANIFEST_NAME
    if not path.is_file():
        source_path = app_root / "parser-backends" / BACKEND_MANIFEST_NAME
        if source_path.is_file():
            path = source_path
    payload = json.loads(path.read_text(encoding="utf-8"))
    if payload.get("manifest_version") != BACKEND_MANIFEST_VERSION:
        raise ValueError("Unsupported bundled backend manifest version.")
    if not isinstance(payload.get("backends"), list):
        raise ValueError("Bundled backend manifest must contain a backends list.")
    return payload


def resolve_backend_asset(relative_path: str, app_root: Path | None = None) -> Path:
    """Resolve one manifest-owned backend asset without allowing path escape."""
    root = backend_asset_root(app_root).resolve()
    path = (root / relative_path).resolve()
    if path != root and root not in path.parents:
        raise ValueError("Backend asset path escapes the bundled backend directory.")
    return path
