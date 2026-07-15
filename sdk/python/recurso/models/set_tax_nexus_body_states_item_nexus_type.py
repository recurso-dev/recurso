from enum import Enum


class SetTaxNexusBodyStatesItemNexusType(str, Enum):
    ECONOMIC = "economic"
    PHYSICAL = "physical"
    VOLUNTARY = "voluntary"

    def __str__(self) -> str:
        return str(self.value)
