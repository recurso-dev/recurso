from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="InitiateCheckoutPaymentResponse200Data")


@_attrs_define
class InitiateCheckoutPaymentResponse200Data:
    """
    Attributes:
        order_id (str | Unset):
        amount (int | Unset): Amount in the lowest currency unit.
        currency (str | Unset):
        invoice_id (UUID | Unset):
        invoice_number (str | Unset):
    """

    order_id: str | Unset = UNSET
    amount: int | Unset = UNSET
    currency: str | Unset = UNSET
    invoice_id: UUID | Unset = UNSET
    invoice_number: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        order_id = self.order_id

        amount = self.amount

        currency = self.currency

        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        invoice_number = self.invoice_number

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if order_id is not UNSET:
            field_dict["order_id"] = order_id
        if amount is not UNSET:
            field_dict["amount"] = amount
        if currency is not UNSET:
            field_dict["currency"] = currency
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if invoice_number is not UNSET:
            field_dict["invoice_number"] = invoice_number

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        order_id = d.pop("order_id", UNSET)

        amount = d.pop("amount", UNSET)

        currency = d.pop("currency", UNSET)

        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        invoice_number = d.pop("invoice_number", UNSET)

        initiate_checkout_payment_response_200_data = cls(
            order_id=order_id,
            amount=amount,
            currency=currency,
            invoice_id=invoice_id,
            invoice_number=invoice_number,
        )

        initiate_checkout_payment_response_200_data.additional_properties = d
        return initiate_checkout_payment_response_200_data

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
