from enum import Enum


class QueryUsageResponse200Granularity(str, Enum):
    DAY = "day"
    MONTH = "month"

    def __str__(self) -> str:
        return str(self.value)
