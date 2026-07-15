from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PortalConfirmPaymentMethodResponse200DataCard")


@_attrs_define
class PortalConfirmPaymentMethodResponse200DataCard:
    """
    Attributes:
        brand (str | Unset):
        last4 (str | Unset):
        exp_month (int | Unset):
        exp_year (int | Unset):
    """

    brand: str | Unset = UNSET
    last4: str | Unset = UNSET
    exp_month: int | Unset = UNSET
    exp_year: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        brand = self.brand

        last4 = self.last4

        exp_month = self.exp_month

        exp_year = self.exp_year

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if brand is not UNSET:
            field_dict["brand"] = brand
        if last4 is not UNSET:
            field_dict["last4"] = last4
        if exp_month is not UNSET:
            field_dict["exp_month"] = exp_month
        if exp_year is not UNSET:
            field_dict["exp_year"] = exp_year

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        brand = d.pop("brand", UNSET)

        last4 = d.pop("last4", UNSET)

        exp_month = d.pop("exp_month", UNSET)

        exp_year = d.pop("exp_year", UNSET)

        portal_confirm_payment_method_response_200_data_card = cls(
            brand=brand,
            last4=last4,
            exp_month=exp_month,
            exp_year=exp_year,
        )

        portal_confirm_payment_method_response_200_data_card.additional_properties = d
        return portal_confirm_payment_method_response_200_data_card

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
