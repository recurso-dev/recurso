from enum import Enum


class HealthResponseStatus(str, Enum):
    DEGRADED = "degraded"
    OK = "ok"

    def __str__(self) -> str:
        return str(self.value)
