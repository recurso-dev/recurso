from enum import Enum


class GetVersionResponse200GatewayMode(str, Enum):
    LIVE = "live"
    NONE = "none"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)
