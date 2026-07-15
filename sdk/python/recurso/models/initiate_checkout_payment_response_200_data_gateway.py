from enum import Enum


class InitiateCheckoutPaymentResponse200DataGateway(str, Enum):
    MOCK = "mock"
    OTHER = "other"
    RAZORPAY = "razorpay"
    STRIPE = "stripe"

    def __str__(self) -> str:
        return str(self.value)
