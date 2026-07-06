from enum import Enum


class EntitlementKind(str, Enum):
    BOOLEAN = "boolean"
    LIMIT = "limit"

    def __str__(self) -> str:
        return str(self.value)
