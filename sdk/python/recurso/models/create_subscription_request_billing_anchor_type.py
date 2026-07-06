from enum import Enum


class CreateSubscriptionRequestBillingAnchorType(str, Enum):
    ACQUISITION = "acquisition"
    FIRST_OF_MONTH = "first_of_month"

    def __str__(self) -> str:
        return str(self.value)
