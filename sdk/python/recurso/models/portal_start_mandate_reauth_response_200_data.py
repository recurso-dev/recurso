from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PortalStartMandateReauthResponse200Data")


@_attrs_define
class PortalStartMandateReauthResponse200Data:
    """
    Attributes:
        auth_url (str | Unset):
        mandate_id (UUID | Unset):
        status (str | Unset):  Example: created.
    """

    auth_url: str | Unset = UNSET
    mandate_id: UUID | Unset = UNSET
    status: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        auth_url = self.auth_url

        mandate_id: str | Unset = UNSET
        if not isinstance(self.mandate_id, Unset):
            mandate_id = str(self.mandate_id)

        status = self.status

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if auth_url is not UNSET:
            field_dict["auth_url"] = auth_url
        if mandate_id is not UNSET:
            field_dict["mandate_id"] = mandate_id
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        auth_url = d.pop("auth_url", UNSET)

        _mandate_id = d.pop("mandate_id", UNSET)
        mandate_id: UUID | Unset
        if isinstance(_mandate_id, Unset):
            mandate_id = UNSET
        else:
            mandate_id = UUID(_mandate_id)

        status = d.pop("status", UNSET)

        portal_start_mandate_reauth_response_200_data = cls(
            auth_url=auth_url,
            mandate_id=mandate_id,
            status=status,
        )

        portal_start_mandate_reauth_response_200_data.additional_properties = d
        return portal_start_mandate_reauth_response_200_data

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
