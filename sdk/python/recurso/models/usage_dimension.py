from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="UsageDimension")


@_attrs_define
class UsageDimension:
    """A dimension-catalog row for the tenant.

    Attributes:
        dimension (str | Unset):
        event_count (int | Unset):
        first_seen (datetime.datetime | Unset):
        last_seen (datetime.datetime | Unset):
    """

    dimension: str | Unset = UNSET
    event_count: int | Unset = UNSET
    first_seen: datetime.datetime | Unset = UNSET
    last_seen: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        dimension = self.dimension

        event_count = self.event_count

        first_seen: str | Unset = UNSET
        if not isinstance(self.first_seen, Unset):
            first_seen = self.first_seen.isoformat()

        last_seen: str | Unset = UNSET
        if not isinstance(self.last_seen, Unset):
            last_seen = self.last_seen.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if dimension is not UNSET:
            field_dict["dimension"] = dimension
        if event_count is not UNSET:
            field_dict["event_count"] = event_count
        if first_seen is not UNSET:
            field_dict["first_seen"] = first_seen
        if last_seen is not UNSET:
            field_dict["last_seen"] = last_seen

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dimension = d.pop("dimension", UNSET)

        event_count = d.pop("event_count", UNSET)

        _first_seen = d.pop("first_seen", UNSET)
        first_seen: datetime.datetime | Unset
        if isinstance(_first_seen, Unset):
            first_seen = UNSET
        else:
            first_seen = datetime.datetime.fromisoformat(_first_seen)

        _last_seen = d.pop("last_seen", UNSET)
        last_seen: datetime.datetime | Unset
        if isinstance(_last_seen, Unset):
            last_seen = UNSET
        else:
            last_seen = datetime.datetime.fromisoformat(_last_seen)

        usage_dimension = cls(
            dimension=dimension,
            event_count=event_count,
            first_seen=first_seen,
            last_seen=last_seen,
        )

        usage_dimension.additional_properties = d
        return usage_dimension

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
