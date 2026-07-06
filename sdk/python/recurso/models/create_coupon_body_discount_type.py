from enum import Enum


class CreateCouponBodyDiscountType(str, Enum):
    AMOUNT = "amount"
    PERCENT = "percent"

    def __str__(self) -> str:
        return str(self.value)
