from enum import Enum


class CheckoutSuccessResponse200DataStatus(str, Enum):
    FAILED = "failed"
    OPEN = "open"
    PAID = "paid"
    PAST_DUE = "past_due"
    PROCESSING = "processing"

    def __str__(self) -> str:
        return str(self.value)
