from enum import Enum


class EntitlementInputKind(str, Enum):
    BOOLEAN = "boolean"
    LIMIT = "limit"

    def __str__(self) -> str:
        return str(self.value)
