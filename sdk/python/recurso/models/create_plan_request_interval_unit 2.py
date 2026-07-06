from enum import Enum


class CreatePlanRequestIntervalUnit(str, Enum):
    DAY = "day"
    MONTH = "month"
    WEEK = "week"
    YEAR = "year"

    def __str__(self) -> str:
        return str(self.value)
