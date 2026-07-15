from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MfaSetupResponse200")


@_attrs_define
class MfaSetupResponse200:
    """
    Attributes:
        secret (str | Unset):
        otpauth_url (str | Unset):
    """

    secret: str | Unset = UNSET
    otpauth_url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        secret = self.secret

        otpauth_url = self.otpauth_url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if secret is not UNSET:
            field_dict["secret"] = secret
        if otpauth_url is not UNSET:
            field_dict["otpauth_url"] = otpauth_url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        secret = d.pop("secret", UNSET)

        otpauth_url = d.pop("otpauth_url", UNSET)

        mfa_setup_response_200 = cls(
            secret=secret,
            otpauth_url=otpauth_url,
        )

        mfa_setup_response_200.additional_properties = d
        return mfa_setup_response_200

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
