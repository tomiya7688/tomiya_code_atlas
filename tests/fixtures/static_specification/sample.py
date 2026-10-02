"""Small source used to check generated symbol references."""


def render(value: str) -> str:
    return normalize(value)


def normalize(value: str) -> str:
    return value.strip()
