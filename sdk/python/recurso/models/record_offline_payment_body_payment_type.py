from enum import Enum


class RecordOfflinePaymentBodyPaymentType(str, Enum):
    BANK_TRANSFER = "bank_transfer"
    CASH = "cash"
    CHEQUE = "cheque"

    def __str__(self) -> str:
        return str(self.value)
