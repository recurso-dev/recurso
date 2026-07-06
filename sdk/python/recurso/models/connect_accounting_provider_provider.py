from enum import Enum


class ConnectAccountingProviderProvider(str, Enum):
    QUICKBOOKS = "quickbooks"
    XERO = "xero"

    def __str__(self) -> str:
        return str(self.value)
