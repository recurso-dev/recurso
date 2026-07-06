from enum import Enum


class CouponDiscountType(str, Enum):
    AMOUNT = "amount"
    PERCENT = "percent"

    def __str__(self) -> str:
        return str(self.value)
