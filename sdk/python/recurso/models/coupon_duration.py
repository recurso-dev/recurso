from enum import Enum


class CouponDuration(str, Enum):
    FOREVER = "forever"
    ONCE = "once"
    REPEATING = "repeating"

    def __str__(self) -> str:
        return str(self.value)
