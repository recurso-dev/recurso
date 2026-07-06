from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="RecordUsageEventBody")


@_attrs_define
class RecordUsageEventBody:
    """
    Attributes:
        subscription_id (UUID):
        customer_id (UUID):
        dimension (str):
        quantity (int):
    """

    subscription_id: UUID
    customer_id: UUID
    dimension: str
    quantity: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        subscription_id = str(self.subscription_id)

        customer_id = str(self.customer_id)

        dimension = self.dimension

        quantity = self.quantity

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "subscription_id": subscription_id,
                "customer_id": customer_id,
                "dimension": dimension,
                "quantity": quantity,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        subscription_id = UUID(d.pop("subscription_id"))

        customer_id = UUID(d.pop("customer_id"))

        dimension = d.pop("dimension")

        quantity = d.pop("quantity")

        record_usage_event_body = cls(
            subscription_id=subscription_id,
            customer_id=customer_id,
            dimension=dimension,
            quantity=quantity,
        )

        record_usage_event_body.additional_properties = d
        return record_usage_event_body

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
