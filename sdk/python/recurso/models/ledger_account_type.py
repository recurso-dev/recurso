from enum import Enum


class LedgerAccountType(str, Enum):
    ASSET = "asset"
    EQUITY = "equity"
    EXPENSE = "expense"
    LIABILITY = "liability"
    REVENUE = "revenue"

    def __str__(self) -> str:
        return str(self.value)
