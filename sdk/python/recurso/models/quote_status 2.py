from enum import Enum


class QuoteStatus(str, Enum):
    ACCEPTED = "accepted"
    DECLINED = "declined"
    DRAFT = "draft"
    EXPIRED = "expired"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)
