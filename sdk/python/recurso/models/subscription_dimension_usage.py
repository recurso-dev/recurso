from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="SubscriptionDimensionUsage")


@_attrs_define
class SubscriptionDimensionUsage:
    """One dimension's usage for a subscription. limit_value/remaining are populated only when the customer holds an
    entitlement limit for a feature_key equal to the dimension name.

        Attributes:
            dimension (str | Unset):
            period_quantity (int | Unset): Total inside the current billing period.
            lifetime_quantity (int | Unset):
            limit_value (int | None | Unset): Effective entitlement limit, or null when none exists.
            remaining (int | None | Unset): limit_value - period_quantity (negative when over the limit); null when
                limit_value is null.
    """

    dimension: str | Unset = UNSET
    period_quantity: int | Unset = UNSET
    lifetime_quantity: int | Unset = UNSET
    limit_value: int | None | Unset = UNSET
    remaining: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        dimension = self.dimension

        period_quantity = self.period_quantity

        lifetime_quantity = self.lifetime_quantity

        limit_value: int | None | Unset
        if isinstance(self.limit_value, Unset):
            limit_value = UNSET
        else:
            limit_value = self.limit_value

        remaining: int | None | Unset
        if isinstance(self.remaining, Unset):
            remaining = UNSET
        else:
            remaining = self.remaining

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if dimension is not UNSET:
            field_dict["dimension"] = dimension
        if period_quantity is not UNSET:
            field_dict["period_quantity"] = period_quantity
        if lifetime_quantity is not UNSET:
            field_dict["lifetime_quantity"] = lifetime_quantity
        if limit_value is not UNSET:
            field_dict["limit_value"] = limit_value
        if remaining is not UNSET:
            field_dict["remaining"] = remaining

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dimension = d.pop("dimension", UNSET)

        period_quantity = d.pop("period_quantity", UNSET)

        lifetime_quantity = d.pop("lifetime_quantity", UNSET)

        def _parse_limit_value(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        limit_value = _parse_limit_value(d.pop("limit_value", UNSET))

        def _parse_remaining(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        remaining = _parse_remaining(d.pop("remaining", UNSET))

        subscription_dimension_usage = cls(
            dimension=dimension,
            period_quantity=period_quantity,
            lifetime_quantity=lifetime_quantity,
            limit_value=limit_value,
            remaining=remaining,
        )

        subscription_dimension_usage.additional_properties = d
        return subscription_dimension_usage

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
