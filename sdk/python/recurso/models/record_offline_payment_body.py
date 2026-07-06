from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.record_offline_payment_body_payment_type import RecordOfflinePaymentBodyPaymentType
from ..types import UNSET, Unset

T = TypeVar("T", bound="RecordOfflinePaymentBody")


@_attrs_define
class RecordOfflinePaymentBody:
    """
    Attributes:
        customer_id (UUID):
        payment_type (RecordOfflinePaymentBodyPaymentType):
        amount (int): Amount in the lowest currency unit.
        invoice_id (UUID | Unset):
        currency (str | Unset):  Default: 'INR'.
        reference_number (str | Unset):
        notes (str | Unset):
        recorded_by (str | Unset):
    """

    customer_id: UUID
    payment_type: RecordOfflinePaymentBodyPaymentType
    amount: int
    invoice_id: UUID | Unset = UNSET
    currency: str | Unset = "INR"
    reference_number: str | Unset = UNSET
    notes: str | Unset = UNSET
    recorded_by: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        customer_id = str(self.customer_id)

        payment_type = self.payment_type.value

        amount = self.amount

        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        currency = self.currency

        reference_number = self.reference_number

        notes = self.notes

        recorded_by = self.recorded_by

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "customer_id": customer_id,
                "payment_type": payment_type,
                "amount": amount,
            }
        )
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if currency is not UNSET:
            field_dict["currency"] = currency
        if reference_number is not UNSET:
            field_dict["reference_number"] = reference_number
        if notes is not UNSET:
            field_dict["notes"] = notes
        if recorded_by is not UNSET:
            field_dict["recorded_by"] = recorded_by

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        customer_id = UUID(d.pop("customer_id"))

        payment_type = RecordOfflinePaymentBodyPaymentType(d.pop("payment_type"))

        amount = d.pop("amount")

        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        currency = d.pop("currency", UNSET)

        reference_number = d.pop("reference_number", UNSET)

        notes = d.pop("notes", UNSET)

        recorded_by = d.pop("recorded_by", UNSET)

        record_offline_payment_body = cls(
            customer_id=customer_id,
            payment_type=payment_type,
            amount=amount,
            invoice_id=invoice_id,
            currency=currency,
            reference_number=reference_number,
            notes=notes,
            recorded_by=recorded_by,
        )

        record_offline_payment_body.additional_properties = d
        return record_offline_payment_body

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
