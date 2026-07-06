from enum import Enum


class CreateSubscriptionRequestPaymentTerms(str, Enum):
    NET0 = "net0"
    NET15 = "net15"
    NET30 = "net30"
    NET60 = "net60"

    def __str__(self) -> str:
        return str(self.value)
