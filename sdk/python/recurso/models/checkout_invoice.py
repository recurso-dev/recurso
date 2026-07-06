from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CheckoutInvoice")


@_attrs_define
class CheckoutInvoice:
    """
    Attributes:
        id (UUID | Unset):
        invoice_number (str | Unset):
        status (str | Unset):
        currency (str | Unset):
        subtotal (int | Unset):
        tax_amount (int | Unset):
        total (int | Unset):
        display_amount (str | Unset): Human-readable major-unit amount (e.g. "118.00").
        due_date (datetime.date | Unset):
        customer_id (UUID | Unset):
    """

    id: UUID | Unset = UNSET
    invoice_number: str | Unset = UNSET
    status: str | Unset = UNSET
    currency: str | Unset = UNSET
    subtotal: int | Unset = UNSET
    tax_amount: int | Unset = UNSET
    total: int | Unset = UNSET
    display_amount: str | Unset = UNSET
    due_date: datetime.date | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        invoice_number = self.invoice_number

        status = self.status

        currency = self.currency

        subtotal = self.subtotal

        tax_amount = self.tax_amount

        total = self.total

        display_amount = self.display_amount

        due_date: str | Unset = UNSET
        if not isinstance(self.due_date, Unset):
            due_date = self.due_date.isoformat()

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if invoice_number is not UNSET:
            field_dict["invoice_number"] = invoice_number
        if status is not UNSET:
            field_dict["status"] = status
        if currency is not UNSET:
            field_dict["currency"] = currency
        if subtotal is not UNSET:
            field_dict["subtotal"] = subtotal
        if tax_amount is not UNSET:
            field_dict["tax_amount"] = tax_amount
        if total is not UNSET:
            field_dict["total"] = total
        if display_amount is not UNSET:
            field_dict["display_amount"] = display_amount
        if due_date is not UNSET:
            field_dict["due_date"] = due_date
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id

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

        invoice_number = d.pop("invoice_number", UNSET)

        status = d.pop("status", UNSET)

        currency = d.pop("currency", UNSET)

        subtotal = d.pop("subtotal", UNSET)

        tax_amount = d.pop("tax_amount", UNSET)

        total = d.pop("total", UNSET)

        display_amount = d.pop("display_amount", UNSET)

        _due_date = d.pop("due_date", UNSET)
        due_date: datetime.date | Unset
        if isinstance(_due_date, Unset):
            due_date = UNSET
        else:
            due_date = datetime.date.fromisoformat(_due_date)

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        checkout_invoice = cls(
            id=id,
            invoice_number=invoice_number,
            status=status,
            currency=currency,
            subtotal=subtotal,
            tax_amount=tax_amount,
            total=total,
            display_amount=display_amount,
            due_date=due_date,
            customer_id=customer_id,
        )

        checkout_invoice.additional_properties = d
        return checkout_invoice

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
