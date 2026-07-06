from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateCancelFlowBody")


@_attrs_define
class CreateCancelFlowBody:
    """
    Attributes:
        name (str):
        is_default (bool | Unset):
        cooldown_days (int | Unset): Days a customer must wait after accepting an offer before re-entering the flow.
            Default: 30.
    """

    name: str
    is_default: bool | Unset = UNSET
    cooldown_days: int | Unset = 30
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        is_default = self.is_default

        cooldown_days = self.cooldown_days

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if is_default is not UNSET:
            field_dict["is_default"] = is_default
        if cooldown_days is not UNSET:
            field_dict["cooldown_days"] = cooldown_days

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        is_default = d.pop("is_default", UNSET)

        cooldown_days = d.pop("cooldown_days", UNSET)

        create_cancel_flow_body = cls(
            name=name,
            is_default=is_default,
            cooldown_days=cooldown_days,
        )

        create_cancel_flow_body.additional_properties = d
        return create_cancel_flow_body

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
