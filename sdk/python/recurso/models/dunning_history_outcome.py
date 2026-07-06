from enum import Enum


class DunningHistoryOutcome(str, Enum):
    FAILURE = "failure"
    SUCCESS = "success"

    def __str__(self) -> str:
        return str(self.value)
