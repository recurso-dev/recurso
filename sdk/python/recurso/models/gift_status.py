from enum import Enum


class GiftStatus(str, Enum):
    PURCHASED = "purchased"
    REDEEMED = "redeemed"

    def __str__(self) -> str:
        return str(self.value)
