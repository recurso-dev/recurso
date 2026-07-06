from enum import Enum


class FXSnapshotSource(str, Enum):
    LIVE = "live"
    STATIC_FALLBACK = "static-fallback"

    def __str__(self) -> str:
        return str(self.value)
