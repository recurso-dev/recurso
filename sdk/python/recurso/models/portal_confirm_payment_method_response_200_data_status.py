from enum import Enum


class PortalConfirmPaymentMethodResponse200DataStatus(str, Enum):
    PROCESSING = "processing"
    SAVED = "saved"

    def __str__(self) -> str:
        return str(self.value)
