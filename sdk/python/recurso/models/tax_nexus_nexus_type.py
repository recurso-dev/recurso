from enum import Enum


class TaxNexusNexusType(str, Enum):
    ECONOMIC = "economic"
    PHYSICAL = "physical"
    VOLUNTARY = "voluntary"

    def __str__(self) -> str:
        return str(self.value)
