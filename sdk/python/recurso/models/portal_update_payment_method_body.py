from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="PortalUpdatePaymentMethodBody")


@_attrs_define
class PortalUpdatePaymentMethodBody:
    """
    Attributes:
        card_brand (str):  Example: visa.
        card_last4 (str):  Example: 4242.
        card_exp_month (int):
        card_exp_year (int):
    """

    card_brand: str
    card_last4: str
    card_exp_month: int
    card_exp_year: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        card_brand = self.card_brand

        card_last4 = self.card_last4

        card_exp_month = self.card_exp_month

        card_exp_year = self.card_exp_year

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "card_brand": card_brand,
                "card_last4": card_last4,
                "card_exp_month": card_exp_month,
                "card_exp_year": card_exp_year,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        card_brand = d.pop("card_brand")

        card_last4 = d.pop("card_last4")

        card_exp_month = d.pop("card_exp_month")

        card_exp_year = d.pop("card_exp_year")

        portal_update_payment_method_body = cls(
            card_brand=card_brand,
            card_last4=card_last4,
            card_exp_month=card_exp_month,
            card_exp_year=card_exp_year,
        )

        portal_update_payment_method_body.additional_properties = d
        return portal_update_payment_method_body

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
