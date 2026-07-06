from enum import Enum


class MandateStatus(str, Enum):
    ACTIVE = "active"
    AUTHORIZED = "authorized"
    CREATED = "created"
    PAUSED = "paused"
    REVOKED = "revoked"

    def __str__(self) -> str:
        return str(self.value)
