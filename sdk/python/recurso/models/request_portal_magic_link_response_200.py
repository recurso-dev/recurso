from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RequestPortalMagicLinkResponse200")


@_attrs_define
class RequestPortalMagicLinkResponse200:
    """
    Attributes:
        message (str | Unset):
        field_dev_link (str | Unset): Verification link — exposed only when APP_ENV=development.
    """

    message: str | Unset = UNSET
    field_dev_link: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        message = self.message

        field_dev_link = self.field_dev_link

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if message is not UNSET:
            field_dict["message"] = message
        if field_dev_link is not UNSET:
            field_dict["_dev_link"] = field_dev_link

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        message = d.pop("message", UNSET)

        field_dev_link = d.pop("_dev_link", UNSET)

        request_portal_magic_link_response_200 = cls(
            message=message,
            field_dev_link=field_dev_link,
        )

        request_portal_magic_link_response_200.additional_properties = d
        return request_portal_magic_link_response_200

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
