from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PostAuthSamlTenantIDAcsBody")


@_attrs_define
class PostAuthSamlTenantIDAcsBody:
    """
    Attributes:
        saml_response (str | Unset):
        relay_state (str | Unset):
    """

    saml_response: str | Unset = UNSET
    relay_state: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        saml_response = self.saml_response

        relay_state = self.relay_state

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if saml_response is not UNSET:
            field_dict["SAMLResponse"] = saml_response
        if relay_state is not UNSET:
            field_dict["RelayState"] = relay_state

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        saml_response = d.pop("SAMLResponse", UNSET)

        relay_state = d.pop("RelayState", UNSET)

        post_auth_saml_tenant_id_acs_body = cls(
            saml_response=saml_response,
            relay_state=relay_state,
        )

        post_auth_saml_tenant_id_acs_body.additional_properties = d
        return post_auth_saml_tenant_id_acs_body

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
