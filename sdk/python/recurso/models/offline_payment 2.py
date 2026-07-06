from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.offline_payment_payment_type import OfflinePaymentPaymentType
from ..types import UNSET, Unset

T = TypeVar("T", bound="OfflinePayment")


@_attrs_define
class OfflinePayment:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        invoice_id (None | Unset | UUID):
        payment_type (OfflinePaymentPaymentType | Unset):
        amount (int | Unset):
        currency (str | Unset):
        reference_number (str | Unset):
        notes (str | Unset):
        recorded_by (str | Unset):
        recorded_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    invoice_id: None | Unset | UUID = UNSET
    payment_type: OfflinePaymentPaymentType | Unset = UNSET
    amount: int | Unset = UNSET
    currency: str | Unset = UNSET
    reference_number: str | Unset = UNSET
    notes: str | Unset = UNSET
    recorded_by: str | Unset = UNSET
    recorded_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        invoice_id: None | str | Unset
        if isinstance(self.invoice_id, Unset):
            invoice_id = UNSET
        elif isinstance(self.invoice_id, UUID):
            invoice_id = str(self.invoice_id)
        else:
            invoice_id = self.invoice_id

        payment_type: str | Unset = UNSET
        if not isinstance(self.payment_type, Unset):
            payment_type = self.payment_type.value

        amount = self.amount

        currency = self.currency

        reference_number = self.reference_number

        notes = self.notes

        recorded_by = self.recorded_by

        recorded_at: str | Unset = UNSET
        if not isinstance(self.recorded_at, Unset):
            recorded_at = self.recorded_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if payment_type is not UNSET:
            field_dict["payment_type"] = payment_type
        if amount is not UNSET:
            field_dict["amount"] = amount
        if currency is not UNSET:
            field_dict["currency"] = currency
        if reference_number is not UNSET:
            field_dict["reference_number"] = reference_number
        if notes is not UNSET:
            field_dict["notes"] = notes
        if recorded_by is not UNSET:
            field_dict["recorded_by"] = recorded_by
        if recorded_at is not UNSET:
            field_dict["recorded_at"] = recorded_at

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

        _tenant_id = d.pop("tenant_id", UNSET)
        tenant_id: UUID | Unset
        if isinstance(_tenant_id, Unset):
            tenant_id = UNSET
        else:
            tenant_id = UUID(_tenant_id)

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

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

        _payment_type = d.pop("payment_type", UNSET)
        payment_type: OfflinePaymentPaymentType | Unset
        if isinstance(_payment_type, Unset):
            payment_type = UNSET
        else:
            payment_type = OfflinePaymentPaymentType(_payment_type)

        amount = d.pop("amount", UNSET)

        currency = d.pop("currency", UNSET)

        reference_number = d.pop("reference_number", UNSET)

        notes = d.pop("notes", UNSET)

        recorded_by = d.pop("recorded_by", UNSET)

        _recorded_at = d.pop("recorded_at", UNSET)
        recorded_at: datetime.datetime | Unset
        if isinstance(_recorded_at, Unset):
            recorded_at = UNSET
        else:
            recorded_at = datetime.datetime.fromisoformat(_recorded_at)

        offline_payment = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            invoice_id=invoice_id,
            payment_type=payment_type,
            amount=amount,
            currency=currency,
            reference_number=reference_number,
            notes=notes,
            recorded_by=recorded_by,
            recorded_at=recorded_at,
        )

        offline_payment.additional_properties = d
        return offline_payment

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
