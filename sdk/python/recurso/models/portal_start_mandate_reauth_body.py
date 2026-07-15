from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PortalStartMandateReauthBody")


@_attrs_define
class PortalStartMandateReauthBody:
    """
    Attributes:
        vpa (str | Unset): Optional UPI id — Razorpay's hosted page collects it when omitted.
    """

    vpa: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        vpa = self.vpa

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if vpa is not UNSET:
            field_dict["vpa"] = vpa

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        vpa = d.pop("vpa", UNSET)

        portal_start_mandate_reauth_body = cls(
            vpa=vpa,
        )

        portal_start_mandate_reauth_body.additional_properties = d
        return portal_start_mandate_reauth_body

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
