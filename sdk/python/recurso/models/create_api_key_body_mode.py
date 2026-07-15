from enum import Enum


class CreateAPIKeyBodyMode(str, Enum):
    LIVE = "live"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)
