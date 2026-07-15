from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="LoginMFABody")


@_attrs_define
class LoginMFABody:
    """
    Attributes:
        mfa_token (str): The single-use challenge token from /auth/login.
        code (str): A current TOTP code or an unused backup code.
    """

    mfa_token: str
    code: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        mfa_token = self.mfa_token

        code = self.code

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "mfa_token": mfa_token,
                "code": code,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mfa_token = d.pop("mfa_token")

        code = d.pop("code")

        login_mfa_body = cls(
            mfa_token=mfa_token,
            code=code,
        )

        login_mfa_body.additional_properties = d
        return login_mfa_body

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
