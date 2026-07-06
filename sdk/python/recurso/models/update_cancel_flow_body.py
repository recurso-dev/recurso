from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="UpdateCancelFlowBody")


@_attrs_define
class UpdateCancelFlowBody:
    """
    Attributes:
        name (str | Unset):
        is_active (bool | Unset):
        is_default (bool | Unset):
        cooldown_days (int | Unset):
    """

    name: str | Unset = UNSET
    is_active: bool | Unset = UNSET
    is_default: bool | Unset = UNSET
    cooldown_days: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        is_active = self.is_active

        is_default = self.is_default

        cooldown_days = self.cooldown_days

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if name is not UNSET:
            field_dict["name"] = name
        if is_active is not UNSET:
            field_dict["is_active"] = is_active
        if is_default is not UNSET:
            field_dict["is_default"] = is_default
        if cooldown_days is not UNSET:
            field_dict["cooldown_days"] = cooldown_days

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name", UNSET)

        is_active = d.pop("is_active", UNSET)

        is_default = d.pop("is_default", UNSET)

        cooldown_days = d.pop("cooldown_days", UNSET)

        update_cancel_flow_body = cls(
            name=name,
            is_active=is_active,
            is_default=is_default,
            cooldown_days=cooldown_days,
        )

        update_cancel_flow_body.additional_properties = d
        return update_cancel_flow_body

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
