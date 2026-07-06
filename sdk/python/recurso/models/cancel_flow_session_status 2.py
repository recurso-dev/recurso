from enum import Enum


class CancelFlowSessionStatus(str, Enum):
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    IN_PROGRESS = "in_progress"
    SAVED = "saved"

    def __str__(self) -> str:
        return str(self.value)
