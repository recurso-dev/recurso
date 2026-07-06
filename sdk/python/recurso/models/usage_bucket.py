from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="UsageBucket")


@_attrs_define
class UsageBucket:
    """One date_trunc'd time bucket of aggregated usage.

    Attributes:
        period (datetime.datetime | Unset): Bucket start (date_trunc of the event timestamps).
        dimension (str | Unset):
        quantity (int | Unset):
    """

    period: datetime.datetime | Unset = UNSET
    dimension: str | Unset = UNSET
    quantity: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        period: str | Unset = UNSET
        if not isinstance(self.period, Unset):
            period = self.period.isoformat()

        dimension = self.dimension

        quantity = self.quantity

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if period is not UNSET:
            field_dict["period"] = period
        if dimension is not UNSET:
            field_dict["dimension"] = dimension
        if quantity is not UNSET:
            field_dict["quantity"] = quantity

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _period = d.pop("period", UNSET)
        period: datetime.datetime | Unset
        if isinstance(_period, Unset):
            period = UNSET
        else:
            period = datetime.datetime.fromisoformat(_period)

        dimension = d.pop("dimension", UNSET)

        quantity = d.pop("quantity", UNSET)

        usage_bucket = cls(
            period=period,
            dimension=dimension,
            quantity=quantity,
        )

        usage_bucket.additional_properties = d
        return usage_bucket

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
