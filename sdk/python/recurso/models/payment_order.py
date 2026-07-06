from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PaymentOrder")


@_attrs_define
class PaymentOrder:
    """Gateway payment order. Field names are capitalized because the gateway port struct is serialized verbatim.

    Attributes:
        id (str | Unset): Gateway order ID.
        amount (int | Unset): Amount in the lowest currency unit.
        currency (str | Unset):
        receipt (str | Unset): Receipt reference (the invoice number).
    """

    id: str | Unset = UNSET
    amount: int | Unset = UNSET
    currency: str | Unset = UNSET
    receipt: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        amount = self.amount

        currency = self.currency

        receipt = self.receipt

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["ID"] = id
        if amount is not UNSET:
            field_dict["Amount"] = amount
        if currency is not UNSET:
            field_dict["Currency"] = currency
        if receipt is not UNSET:
            field_dict["Receipt"] = receipt

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("ID", UNSET)

        amount = d.pop("Amount", UNSET)

        currency = d.pop("Currency", UNSET)

        receipt = d.pop("Receipt", UNSET)

        payment_order = cls(
            id=id,
            amount=amount,
            currency=currency,
            receipt=receipt,
        )

        payment_order.additional_properties = d
        return payment_order

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
