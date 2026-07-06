from enum import Enum


class MandateFrequency(str, Enum):
    MONTHLY = "monthly"
    QUARTERLY = "quarterly"
    WEEKLY = "weekly"
    YEARLY = "yearly"

    def __str__(self) -> str:
        return str(self.value)
