from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PortalStartPaymentMethodSetupResponse200Data")


@_attrs_define
class PortalStartPaymentMethodSetupResponse200Data:
    """
    Attributes:
        client_secret (str | Unset):
        publishable_key (str | Unset):
    """

    client_secret: str | Unset = UNSET
    publishable_key: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        client_secret = self.client_secret

        publishable_key = self.publishable_key

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if client_secret is not UNSET:
            field_dict["client_secret"] = client_secret
        if publishable_key is not UNSET:
            field_dict["publishable_key"] = publishable_key

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        client_secret = d.pop("client_secret", UNSET)

        publishable_key = d.pop("publishable_key", UNSET)

        portal_start_payment_method_setup_response_200_data = cls(
            client_secret=client_secret,
            publishable_key=publishable_key,
        )

        portal_start_payment_method_setup_response_200_data.additional_properties = d
        return portal_start_payment_method_setup_response_200_data

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
