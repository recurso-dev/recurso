from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="SSOConnectionUpsertRequest")


@_attrs_define
class SSOConnectionUpsertRequest:
    """
    Attributes:
        idp_entity_id (str | Unset):
        idp_sso_url (str | Unset):
        idp_certificate (str | Unset):
        idp_metadata_xml (str | Unset):
        enabled (bool | Unset):
    """

    idp_entity_id: str | Unset = UNSET
    idp_sso_url: str | Unset = UNSET
    idp_certificate: str | Unset = UNSET
    idp_metadata_xml: str | Unset = UNSET
    enabled: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        idp_entity_id = self.idp_entity_id

        idp_sso_url = self.idp_sso_url

        idp_certificate = self.idp_certificate

        idp_metadata_xml = self.idp_metadata_xml

        enabled = self.enabled

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if idp_entity_id is not UNSET:
            field_dict["idp_entity_id"] = idp_entity_id
        if idp_sso_url is not UNSET:
            field_dict["idp_sso_url"] = idp_sso_url
        if idp_certificate is not UNSET:
            field_dict["idp_certificate"] = idp_certificate
        if idp_metadata_xml is not UNSET:
            field_dict["idp_metadata_xml"] = idp_metadata_xml
        if enabled is not UNSET:
            field_dict["enabled"] = enabled

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        idp_entity_id = d.pop("idp_entity_id", UNSET)

        idp_sso_url = d.pop("idp_sso_url", UNSET)

        idp_certificate = d.pop("idp_certificate", UNSET)

        idp_metadata_xml = d.pop("idp_metadata_xml", UNSET)

        enabled = d.pop("enabled", UNSET)

        sso_connection_upsert_request = cls(
            idp_entity_id=idp_entity_id,
            idp_sso_url=idp_sso_url,
            idp_certificate=idp_certificate,
            idp_metadata_xml=idp_metadata_xml,
            enabled=enabled,
        )

        sso_connection_upsert_request.additional_properties = d
        return sso_connection_upsert_request

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
