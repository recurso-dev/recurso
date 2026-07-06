from enum import Enum


class CreditNoteStatus(str, Enum):
    ISSUED = "issued"
    USED = "used"
    VOID = "void"

    def __str__(self) -> str:
        return str(self.value)
