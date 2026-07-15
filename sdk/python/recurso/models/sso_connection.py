from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="SSOConnection")


@_attrs_define
class SSOConnection:
    """
    Attributes:
        tenant_id (UUID):
        enabled (bool):
        configured (bool): Whether the connection has enough IdP detail to attempt a login.
        idp_entity_id (str | Unset):
        idp_sso_url (str | Unset):
        idp_certificate (str | Unset): Base64/PEM X.509 signing certificate.
        idp_metadata_xml (str | Unset): Optional full IdP metadata; takes precedence over the discrete fields.
        sp_metadata_url (str | Unset): Public SP metadata URL to hand to the IdP.
        sp_acs_url (str | Unset): Public SP Assertion Consumer Service URL.
    """

    tenant_id: UUID
    enabled: bool
    configured: bool
    idp_entity_id: str | Unset = UNSET
    idp_sso_url: str | Unset = UNSET
    idp_certificate: str | Unset = UNSET
    idp_metadata_xml: str | Unset = UNSET
    sp_metadata_url: str | Unset = UNSET
    sp_acs_url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        tenant_id = str(self.tenant_id)

        enabled = self.enabled

        configured = self.configured

        idp_entity_id = self.idp_entity_id

        idp_sso_url = self.idp_sso_url

        idp_certificate = self.idp_certificate

        idp_metadata_xml = self.idp_metadata_xml

        sp_metadata_url = self.sp_metadata_url

        sp_acs_url = self.sp_acs_url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "tenant_id": tenant_id,
                "enabled": enabled,
                "configured": configured,
            }
        )
        if idp_entity_id is not UNSET:
            field_dict["idp_entity_id"] = idp_entity_id
        if idp_sso_url is not UNSET:
            field_dict["idp_sso_url"] = idp_sso_url
        if idp_certificate is not UNSET:
            field_dict["idp_certificate"] = idp_certificate
        if idp_metadata_xml is not UNSET:
            field_dict["idp_metadata_xml"] = idp_metadata_xml
        if sp_metadata_url is not UNSET:
            field_dict["sp_metadata_url"] = sp_metadata_url
        if sp_acs_url is not UNSET:
            field_dict["sp_acs_url"] = sp_acs_url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        tenant_id = UUID(d.pop("tenant_id"))

        enabled = d.pop("enabled")

        configured = d.pop("configured")

        idp_entity_id = d.pop("idp_entity_id", UNSET)

        idp_sso_url = d.pop("idp_sso_url", UNSET)

        idp_certificate = d.pop("idp_certificate", UNSET)

        idp_metadata_xml = d.pop("idp_metadata_xml", UNSET)

        sp_metadata_url = d.pop("sp_metadata_url", UNSET)

        sp_acs_url = d.pop("sp_acs_url", UNSET)

        sso_connection = cls(
            tenant_id=tenant_id,
            enabled=enabled,
            configured=configured,
            idp_entity_id=idp_entity_id,
            idp_sso_url=idp_sso_url,
            idp_certificate=idp_certificate,
            idp_metadata_xml=idp_metadata_xml,
            sp_metadata_url=sp_metadata_url,
            sp_acs_url=sp_acs_url,
        )

        sso_connection.additional_properties = d
        return sso_connection

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
