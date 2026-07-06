from enum import Enum


class CancelFlowStepType(str, Enum):
    CONFIRMATION = "confirmation"
    OFFER = "offer"
    SURVEY = "survey"

    def __str__(self) -> str:
        return str(self.value)
