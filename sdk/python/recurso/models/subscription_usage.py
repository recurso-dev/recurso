from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.subscription_dimension_usage import SubscriptionDimensionUsage


T = TypeVar("T", bound="SubscriptionUsage")


@_attrs_define
class SubscriptionUsage:
    """Per-subscription usage report (GET /v1/subscriptions/{id}/usage).

    Attributes:
        subscription_id (UUID | Unset):
        customer_id (UUID | Unset):
        current_period_start (datetime.datetime | Unset):
        current_period_end (datetime.datetime | Unset):
        dimensions (list[SubscriptionDimensionUsage] | Unset):
    """

    subscription_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    current_period_start: datetime.datetime | Unset = UNSET
    current_period_end: datetime.datetime | Unset = UNSET
    dimensions: list[SubscriptionDimensionUsage] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        subscription_id: str | Unset = UNSET
        if not isinstance(self.subscription_id, Unset):
            subscription_id = str(self.subscription_id)

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        current_period_start: str | Unset = UNSET
        if not isinstance(self.current_period_start, Unset):
            current_period_start = self.current_period_start.isoformat()

        current_period_end: str | Unset = UNSET
        if not isinstance(self.current_period_end, Unset):
            current_period_end = self.current_period_end.isoformat()

        dimensions: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.dimensions, Unset):
            dimensions = []
            for dimensions_item_data in self.dimensions:
                dimensions_item = dimensions_item_data.to_dict()
                dimensions.append(dimensions_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if subscription_id is not UNSET:
            field_dict["subscription_id"] = subscription_id
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if current_period_start is not UNSET:
            field_dict["current_period_start"] = current_period_start
        if current_period_end is not UNSET:
            field_dict["current_period_end"] = current_period_end
        if dimensions is not UNSET:
            field_dict["dimensions"] = dimensions

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.subscription_dimension_usage import SubscriptionDimensionUsage

        d = dict(src_dict)
        _subscription_id = d.pop("subscription_id", UNSET)
        subscription_id: UUID | Unset
        if isinstance(_subscription_id, Unset):
            subscription_id = UNSET
        else:
            subscription_id = UUID(_subscription_id)

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        _current_period_start = d.pop("current_period_start", UNSET)
        current_period_start: datetime.datetime | Unset
        if isinstance(_current_period_start, Unset):
            current_period_start = UNSET
        else:
            current_period_start = datetime.datetime.fromisoformat(_current_period_start)

        _current_period_end = d.pop("current_period_end", UNSET)
        current_period_end: datetime.datetime | Unset
        if isinstance(_current_period_end, Unset):
            current_period_end = UNSET
        else:
            current_period_end = datetime.datetime.fromisoformat(_current_period_end)

        _dimensions = d.pop("dimensions", UNSET)
        dimensions: list[SubscriptionDimensionUsage] | Unset = UNSET
        if _dimensions is not UNSET:
            dimensions = []
            for dimensions_item_data in _dimensions:
                dimensions_item = SubscriptionDimensionUsage.from_dict(dimensions_item_data)

                dimensions.append(dimensions_item)

        subscription_usage = cls(
            subscription_id=subscription_id,
            customer_id=customer_id,
            current_period_start=current_period_start,
            current_period_end=current_period_end,
            dimensions=dimensions,
        )

        subscription_usage.additional_properties = d
        return subscription_usage

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
