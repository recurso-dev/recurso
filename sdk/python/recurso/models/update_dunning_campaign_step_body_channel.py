from enum import Enum


class UpdateDunningCampaignStepBodyChannel(str, Enum):
    EMAIL = "email"
    IN_APP = "in_app"
    SMS = "sms"

    def __str__(self) -> str:
        return str(self.value)
