from enum import Enum


class HandleStripeWebhookResponse200Status(str, Enum):
    IGNORED = "ignored"
    OK = "ok"

    def __str__(self) -> str:
        return str(self.value)
