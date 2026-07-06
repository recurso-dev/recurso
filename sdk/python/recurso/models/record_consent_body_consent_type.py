from enum import Enum


class RecordConsentBodyConsentType(str, Enum):
    DATA_PROCESSING = "data_processing"
    EMAIL_MARKETING = "email_marketing"
    PRIVACY_POLICY = "privacy_policy"
    RECURRING_BILLING = "recurring_billing"
    TERMS_OF_SERVICE = "terms_of_service"

    def __str__(self) -> str:
        return str(self.value)
