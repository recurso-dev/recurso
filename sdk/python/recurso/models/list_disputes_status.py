from enum import Enum


class ListDisputesStatus(str, Enum):
    OPEN = "open"
    RESOLVED = "resolved"

    def __str__(self) -> str:
        return str(self.value)
