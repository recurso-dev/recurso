from enum import Enum


class ReferralStatus(str, Enum):
    PENDING = "pending"
    QUALIFIED = "qualified"
    REWARDED = "rewarded"

    def __str__(self) -> str:
        return str(self.value)
