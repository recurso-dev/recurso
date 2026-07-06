from enum import Enum


class AccountingConnectionProvider(str, Enum):
    QUICKBOOKS = "quickbooks"
    XERO = "xero"

    def __str__(self) -> str:
        return str(self.value)
