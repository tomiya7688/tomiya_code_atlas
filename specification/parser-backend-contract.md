# Parser Backend Contract

This specification defines the stable boundary for parser backends used by Tomiya Code Atlas language adapters. It applies to in-process libraries, native bindings, bundled helper executables, and subprocess parsers.

## Contract version

The current contract version is **`1`**.

A backend must expose its own stable `backend_id`, target `language`, hosting `kind`, and `contract_version`. A host must reject an unsupported contract version as `protocol_error` rather than guessing compatibility.

## Boundary rule

A replaceable parser backend must not expose backend-specific AST, syntax-tree, compiler, FFI, or exception types above the language-adapter boundary.

```text
parser library / helper / native code
            ↓
       ParserBackend
            ↓
    Common IR (ModuleIR)
            ↓
Analyzer / Generator / Evaluator
```

A backend may internally use any parser implementation language or library. Conversion from the backend's native representation into Common IR remains inside this boundary.

## Normalized failure kinds

Version 1 defines exactly these stable categories:

- `failure` — generic backend execution/parsing failure that does not fit a more specific category
- `timeout` — host or backend time budget exceeded
- `unsupported_syntax` — syntax or language feature is intentionally unsupported by this backend
- `unsupported_language` — backend does not support the requested source language
- `protocol_error` — malformed message, incompatible version, missing required field, invalid response, or transport contract violation

Each normalized failure contains:

```json
{
  "kind": "failure",
  "message": "human-readable detail",
  "backend_id": "example-python-backend",
  "retryable": false
}
```

Backend-library-specific exception names, stack traces, compiler objects, or FFI handles must not be required by callers above the adapter boundary. Diagnostic details may be logged inside the boundary when safe and useful.

## Subprocess / helper wire protocol

When a parser backend is hosted as a helper process, version 1 uses UTF-8 JSON messages. A transport may frame messages as one request/response per process invocation or as newline-delimited JSON for a persistent helper, but each logical message uses the same envelope.

### Request

```json
{
  "contract_version": "1",
  "request_id": "opaque-request-id",
  "operation": "parse",
  "language": "python",
  "source": "def main(): pass\n",
  "path": "optional/source.py"
}
```

Required fields:
- `contract_version`
- `request_id`
- `operation` (`parse` in version 1)
- `language`
- `source`

`path` is optional metadata and must not be required for parsing source text.

### Success response

```json
{
  "contract_version": "1",
  "request_id": "opaque-request-id",
  "ok": true,
  "ir": {}
}
```

`ir` is the serialized Common IR payload. The helper is responsible for translating its parser-specific structures into the language-neutral IR schema before responding.

### Failure response

```json
{
  "contract_version": "1",
  "request_id": "opaque-request-id",
  "ok": false,
  "error": {
    "kind": "unsupported_syntax",
    "message": "pattern matching is not supported by this backend",
    "backend_id": "example-python-backend",
    "retryable": false
  }
}
```

A response must contain either successful `ir` or a normalized `error`, not both.

## Transport rules

- Protocol payloads are UTF-8.
- Helper stdout is reserved for protocol responses when stdout is the transport; diagnostics go to stderr.
- The host owns process timeout enforcement and converts timeout into the normalized `timeout` category.
- Non-zero exit without a valid normalized response becomes `failure` or `protocol_error`, depending on whether execution failed or the protocol was violated.
- Unknown optional fields may be ignored for forward compatibility.
- Missing required version-1 fields are `protocol_error`.
- Request IDs must be echoed unchanged so persistent helpers can correlate responses.

## Native / in-process backends

Native libraries and in-process parsers do not need to serialize JSON internally. They must nevertheless expose the same logical contract to the language adapter:

- stable backend identity/version metadata
- Common IR output
- normalized failure categories
- no parser-specific types above the boundary

## Distribution

A backend may be bundled with the Windows application as a helper executable or native library. The product must not require a user-installed compiler SDK/runtime/toolchain unless explicitly documented as a separate optional integration.

The packaging layer owns locating bundled binaries/libraries. Analyzer / Generator / Evaluator code must not know their filesystem paths, executable names, FFI handles, or transport mechanism.

## Compatibility

Changing the meaning of required fields, removing a failure kind, or changing the wire envelope requires a new contract version. Additive optional fields may remain within version 1 when older hosts can safely ignore them.
