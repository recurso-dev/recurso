from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="LedgerTransaction")


@_attrs_define
class LedgerTransaction:
    """
    Attributes:
        id (UUID | Unset):
        debit_account_id (UUID | Unset):
        credit_account_id (UUID | Unset):
        amount (int | Unset):
        ledger_id (int | Unset):
        code (int | Unset): Transfer code (1 = invoice posting, 3 = payment posting).
        reference_id (UUID | Unset): Invoice or payment ID the transfer references.
        description (str | Unset):
        timestamp (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    debit_account_id: UUID | Unset = UNSET
    credit_account_id: UUID | Unset = UNSET
    amount: int | Unset = UNSET
    ledger_id: int | Unset = UNSET
    code: int | Unset = UNSET
    reference_id: UUID | Unset = UNSET
    description: str | Unset = UNSET
    timestamp: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        debit_account_id: str | Unset = UNSET
        if not isinstance(self.debit_account_id, Unset):
            debit_account_id = str(self.debit_account_id)

        credit_account_id: str | Unset = UNSET
        if not isinstance(self.credit_account_id, Unset):
            credit_account_id = str(self.credit_account_id)

        amount = self.amount

        ledger_id = self.ledger_id

        code = self.code

        reference_id: str | Unset = UNSET
        if not isinstance(self.reference_id, Unset):
            reference_id = str(self.reference_id)

        description = self.description

        timestamp: str | Unset = UNSET
        if not isinstance(self.timestamp, Unset):
            timestamp = self.timestamp.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if debit_account_id is not UNSET:
            field_dict["debit_account_id"] = debit_account_id
        if credit_account_id is not UNSET:
            field_dict["credit_account_id"] = credit_account_id
        if amount is not UNSET:
            field_dict["amount"] = amount
        if ledger_id is not UNSET:
            field_dict["ledger_id"] = ledger_id
        if code is not UNSET:
            field_dict["code"] = code
        if reference_id is not UNSET:
            field_dict["reference_id"] = reference_id
        if description is not UNSET:
            field_dict["description"] = description
        if timestamp is not UNSET:
            field_dict["timestamp"] = timestamp

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _id = d.pop("id", UNSET)
        id: UUID | Unset
        if isinstance(_id, Unset):
            id = UNSET
        else:
            id = UUID(_id)

        _debit_account_id = d.pop("debit_account_id", UNSET)
        debit_account_id: UUID | Unset
        if isinstance(_debit_account_id, Unset):
            debit_account_id = UNSET
        else:
            debit_account_id = UUID(_debit_account_id)

        _credit_account_id = d.pop("credit_account_id", UNSET)
        credit_account_id: UUID | Unset
        if isinstance(_credit_account_id, Unset):
            credit_account_id = UNSET
        else:
            credit_account_id = UUID(_credit_account_id)

        amount = d.pop("amount", UNSET)

        ledger_id = d.pop("ledger_id", UNSET)

        code = d.pop("code", UNSET)

        _reference_id = d.pop("reference_id", UNSET)
        reference_id: UUID | Unset
        if isinstance(_reference_id, Unset):
            reference_id = UNSET
        else:
            reference_id = UUID(_reference_id)

        description = d.pop("description", UNSET)

        _timestamp = d.pop("timestamp", UNSET)
        timestamp: datetime.datetime | Unset
        if isinstance(_timestamp, Unset):
            timestamp = UNSET
        else:
            timestamp = datetime.datetime.fromisoformat(_timestamp)

        ledger_transaction = cls(
            id=id,
            debit_account_id=debit_account_id,
            credit_account_id=credit_account_id,
            amount=amount,
            ledger_id=ledger_id,
            code=code,
            reference_id=reference_id,
            description=description,
            timestamp=timestamp,
        )

        ledger_transaction.additional_properties = d
        return ledger_transaction

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
