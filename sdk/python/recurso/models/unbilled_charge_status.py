from enum import Enum


class UnbilledChargeStatus(str, Enum):
    CANCELED = "canceled"
    INVOICED = "invoiced"
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)
