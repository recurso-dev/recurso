from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="LineItem")


@_attrs_define
class LineItem:
    """
    Attributes:
        description (str | Unset):
        quantity (int | Unset):
        unit_price (int | Unset): Price per unit in the lowest currency unit.
        amount (int | Unset): quantity x unit_price, in the lowest currency unit.
    """

    description: str | Unset = UNSET
    quantity: int | Unset = UNSET
    unit_price: int | Unset = UNSET
    amount: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        description = self.description

        quantity = self.quantity

        unit_price = self.unit_price

        amount = self.amount

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if description is not UNSET:
            field_dict["description"] = description
        if quantity is not UNSET:
            field_dict["quantity"] = quantity
        if unit_price is not UNSET:
            field_dict["unit_price"] = unit_price
        if amount is not UNSET:
            field_dict["amount"] = amount

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description", UNSET)

        quantity = d.pop("quantity", UNSET)

        unit_price = d.pop("unit_price", UNSET)

        amount = d.pop("amount", UNSET)

        line_item = cls(
            description=description,
            quantity=quantity,
            unit_price=unit_price,
            amount=amount,
        )

        line_item.additional_properties = d
        return line_item

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
