from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="DunningOverview")


@_attrs_define
class DunningOverview:
    """
    Attributes:
        total_retries (int | Unset):
        total_successes (int | Unset):
        success_rate (float | Unset):
    """

    total_retries: int | Unset = UNSET
    total_successes: int | Unset = UNSET
    success_rate: float | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        total_retries = self.total_retries

        total_successes = self.total_successes

        success_rate = self.success_rate

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if total_retries is not UNSET:
            field_dict["total_retries"] = total_retries
        if total_successes is not UNSET:
            field_dict["total_successes"] = total_successes
        if success_rate is not UNSET:
            field_dict["success_rate"] = success_rate

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        total_retries = d.pop("total_retries", UNSET)

        total_successes = d.pop("total_successes", UNSET)

        success_rate = d.pop("success_rate", UNSET)

        dunning_overview = cls(
            total_retries=total_retries,
            total_successes=total_successes,
            success_rate=success_rate,
        )

        dunning_overview.additional_properties = d
        return dunning_overview

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
