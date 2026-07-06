from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="BillingAddress")


@_attrs_define
class BillingAddress:
    """
    Attributes:
        line1 (str | Unset):
        city (str | Unset):
        state (str | Unset):
        zip_ (str | Unset):
        country (str | Unset):
    """

    line1: str | Unset = UNSET
    city: str | Unset = UNSET
    state: str | Unset = UNSET
    zip_: str | Unset = UNSET
    country: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        line1 = self.line1

        city = self.city

        state = self.state

        zip_ = self.zip_

        country = self.country

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if line1 is not UNSET:
            field_dict["line1"] = line1
        if city is not UNSET:
            field_dict["city"] = city
        if state is not UNSET:
            field_dict["state"] = state
        if zip_ is not UNSET:
            field_dict["zip"] = zip_
        if country is not UNSET:
            field_dict["country"] = country

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        line1 = d.pop("line1", UNSET)

        city = d.pop("city", UNSET)

        state = d.pop("state", UNSET)

        zip_ = d.pop("zip", UNSET)

        country = d.pop("country", UNSET)

        billing_address = cls(
            line1=line1,
            city=city,
            state=state,
            zip_=zip_,
            country=country,
        )

        billing_address.additional_properties = d
        return billing_address

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
