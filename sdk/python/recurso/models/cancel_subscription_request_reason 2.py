from enum import Enum


class CancelSubscriptionRequestReason(str, Enum):
    CUSTOMER_SERVICE = "customer_service"
    MISSING_FEATURES = "missing_features"
    NOT_USING = "not_using"
    OTHER = "other"
    SWITCHING_COMPETITOR = "switching_competitor"
    TECHNICAL_ISSUES = "technical_issues"
    TEMPORARY_PAUSE = "temporary_pause"
    TOO_EXPENSIVE = "too_expensive"

    def __str__(self) -> str:
        return str(self.value)
