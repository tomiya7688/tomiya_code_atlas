"""Check the minimum stable CPython patch level required by TLS policy."""

from __future__ import annotations

import sys
from collections.abc import Sequence


MINIMUM_PYTHON_PATCH = {
    (3, 11): 18,
    (3, 12): 16,
    (3, 13): 18,
    (3, 14): 10,
    (3, 15): 2,
}


# {
#   責務: [
#     python_tls_floor_error: CPythonの安定版patch floorへの適合を判定する,
#   ],
#   引数: [
#     version: 判定するPythonのversion_info,
#     release_level: CPython release level,
#     implementation: Python実装名,
#   ],
#   戻り値: [
#     error: 不適合の説明。適合時はNone,
#   ],
# }
def python_tls_floor_error(
    version: Sequence[int],
    release_level: str,
    implementation: str = "cpython",
) -> str | None:
    python_version = tuple(version[:3])
    python_minor = python_version[:2]
    minimum_patch = MINIMUM_PYTHON_PATCH.get(python_minor)
    if implementation != "cpython":
        return "CPython is required for TLS verification."
    if release_level != "final":
        return "A stable CPython release is required for TLS verification."
    if minimum_patch is None:
        return (
            f"Python {python_minor[0]}.{python_minor[1]} has no configured "
            "TLS patch floor."
        )
    if python_version[2] < minimum_patch:
        return (
            f"Python {python_minor[0]}.{python_minor[1]}.{minimum_patch} "
            "or newer is required."
        )
    return None


if __name__ == "__main__":
    error = python_tls_floor_error(
        sys.version_info,
        sys.version_info.releaselevel,
        sys.implementation.name,
    )
    if error:
        print(f"[ERROR] {error}", file=sys.stderr)
        raise SystemExit(1)
    print(
        "CPython "
        f"{sys.version_info.major}.{sys.version_info.minor}."
        f"{sys.version_info.micro} meets the TLS patch floor."
    )
