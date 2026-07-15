from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.create_api_key_body_mode import CreateAPIKeyBodyMode
from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateAPIKeyBody")


@_attrs_define
class CreateAPIKeyBody:
    """
    Attributes:
        name (str | Unset):
        mode (CreateAPIKeyBodyMode | Unset):  Default: CreateAPIKeyBodyMode.TEST.
    """

    name: str | Unset = UNSET
    mode: CreateAPIKeyBodyMode | Unset = CreateAPIKeyBodyMode.TEST
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        mode: str | Unset = UNSET
        if not isinstance(self.mode, Unset):
            mode = self.mode.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if name is not UNSET:
            field_dict["name"] = name
        if mode is not UNSET:
            field_dict["mode"] = mode

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name", UNSET)

        _mode = d.pop("mode", UNSET)
        mode: CreateAPIKeyBodyMode | Unset
        if isinstance(_mode, Unset):
            mode = UNSET
        else:
            mode = CreateAPIKeyBodyMode(_mode)

        create_api_key_body = cls(
            name=name,
            mode=mode,
        )

        create_api_key_body.additional_properties = d
        return create_api_key_body

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
