import pytest

from tools.python_tls_floor import python_tls_floor_error


@pytest.mark.parametrize(
    ("version", "release_level"),
    [
        ((3, 11, 18), "final"),
        ((3, 12, 16), "final"),
        ((3, 13, 18), "final"),
        ((3, 14, 10), "final"),
        ((3, 15, 2), "final"),
    ],
)
def test_python_tls_floor_accepts_minimum_stable_releases(
    version: tuple[int, int, int], release_level: str
) -> None:
    assert python_tls_floor_error(version, release_level) is None


@pytest.mark.parametrize(
    ("version", "release_level", "implementation", "expected_error"),
    [
        (
            (3, 11, 17),
            "final",
            "cpython",
            "Python 3.11.18 or newer is required.",
        ),
        (
            (3, 12, 16),
            "candidate",
            "cpython",
            "A stable CPython release is required for TLS verification.",
        ),
        (
            (3, 16, 0),
            "final",
            "cpython",
            "Python 3.16 has no configured TLS patch floor.",
        ),
        (
            (3, 11, 18),
            "final",
            "pypy",
            "CPython is required for TLS verification.",
        ),
    ],
)
def test_python_tls_floor_reports_unsupported_releases(
    version: tuple[int, int, int],
    release_level: str,
    implementation: str,
    expected_error: str,
) -> None:
    assert (
        python_tls_floor_error(version, release_level, implementation) == expected_error
    )
