from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ReconciliationDiscrepancy")


@_attrs_define
class ReconciliationDiscrepancy:
    """
    Attributes:
        type_ (str | Unset): Discrepancy category (e.g. missing/incorrect invoice or payment posting, orphan
            transaction).
        invoice_id (None | Unset | UUID):
        transaction_id (None | Unset | UUID):
        reference_id (None | Unset | UUID):
        expected_amount (int | Unset):
        found_amount (int | Unset):
    """

    type_: str | Unset = UNSET
    invoice_id: None | Unset | UUID = UNSET
    transaction_id: None | Unset | UUID = UNSET
    reference_id: None | Unset | UUID = UNSET
    expected_amount: int | Unset = UNSET
    found_amount: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        type_ = self.type_

        invoice_id: None | str | Unset
        if isinstance(self.invoice_id, Unset):
            invoice_id = UNSET
        elif isinstance(self.invoice_id, UUID):
            invoice_id = str(self.invoice_id)
        else:
            invoice_id = self.invoice_id

        transaction_id: None | str | Unset
        if isinstance(self.transaction_id, Unset):
            transaction_id = UNSET
        elif isinstance(self.transaction_id, UUID):
            transaction_id = str(self.transaction_id)
        else:
            transaction_id = self.transaction_id

        reference_id: None | str | Unset
        if isinstance(self.reference_id, Unset):
            reference_id = UNSET
        elif isinstance(self.reference_id, UUID):
            reference_id = str(self.reference_id)
        else:
            reference_id = self.reference_id

        expected_amount = self.expected_amount

        found_amount = self.found_amount

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if type_ is not UNSET:
            field_dict["type"] = type_
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if transaction_id is not UNSET:
            field_dict["transaction_id"] = transaction_id
        if reference_id is not UNSET:
            field_dict["reference_id"] = reference_id
        if expected_amount is not UNSET:
            field_dict["expected_amount"] = expected_amount
        if found_amount is not UNSET:
            field_dict["found_amount"] = found_amount

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        type_ = d.pop("type", UNSET)

        def _parse_invoice_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                invoice_id_type_0 = UUID(data)

                return invoice_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        invoice_id = _parse_invoice_id(d.pop("invoice_id", UNSET))

        def _parse_transaction_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                transaction_id_type_0 = UUID(data)

                return transaction_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        transaction_id = _parse_transaction_id(d.pop("transaction_id", UNSET))

        def _parse_reference_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                reference_id_type_0 = UUID(data)

                return reference_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        reference_id = _parse_reference_id(d.pop("reference_id", UNSET))

        expected_amount = d.pop("expected_amount", UNSET)

        found_amount = d.pop("found_amount", UNSET)

        reconciliation_discrepancy = cls(
            type_=type_,
            invoice_id=invoice_id,
            transaction_id=transaction_id,
            reference_id=reference_id,
            expected_amount=expected_amount,
            found_amount=found_amount,
        )

        reconciliation_discrepancy.additional_properties = d
        return reconciliation_discrepancy

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
