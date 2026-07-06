from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="UsageStats")


@_attrs_define
class UsageStats:
    """
    Attributes:
        dimension (str | Unset):
        total_quantity (int | Unset):
    """

    dimension: str | Unset = UNSET
    total_quantity: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        dimension = self.dimension

        total_quantity = self.total_quantity

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if dimension is not UNSET:
            field_dict["dimension"] = dimension
        if total_quantity is not UNSET:
            field_dict["total_quantity"] = total_quantity

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dimension = d.pop("dimension", UNSET)

        total_quantity = d.pop("total_quantity", UNSET)

        usage_stats = cls(
            dimension=dimension,
            total_quantity=total_quantity,
        )

        usage_stats.additional_properties = d
        return usage_stats

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
