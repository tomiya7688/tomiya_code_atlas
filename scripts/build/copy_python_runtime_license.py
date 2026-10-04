"""Copy the exact PSF license file belonging to the interpreter being frozen."""

from __future__ import annotations

from pathlib import Path
import shutil
import sys


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: copy_python_runtime_license.py <destination>")
    source = Path(sys.base_prefix) / "LICENSE.txt"
    if not source.is_file():
        source = Path(sys.base_prefix) / "License.txt"
    if not source.is_file():
        raise SystemExit(f"CPython license file is missing from {sys.base_prefix}")
    destination = Path(sys.argv[1])
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, destination)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
