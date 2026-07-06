from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RedeliverEventResponse202Data")


@_attrs_define
class RedeliverEventResponse202Data:
    """
    Attributes:
        event_id (UUID | Unset):
        deliveries_queued (int | Unset): Number of endpoint deliveries queued (0 when no active endpoint subscribes to
            the event type).
    """

    event_id: UUID | Unset = UNSET
    deliveries_queued: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        event_id: str | Unset = UNSET
        if not isinstance(self.event_id, Unset):
            event_id = str(self.event_id)

        deliveries_queued = self.deliveries_queued

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if event_id is not UNSET:
            field_dict["event_id"] = event_id
        if deliveries_queued is not UNSET:
            field_dict["deliveries_queued"] = deliveries_queued

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _event_id = d.pop("event_id", UNSET)
        event_id: UUID | Unset
        if isinstance(_event_id, Unset):
            event_id = UNSET
        else:
            event_id = UUID(_event_id)

        deliveries_queued = d.pop("deliveries_queued", UNSET)

        redeliver_event_response_202_data = cls(
            event_id=event_id,
            deliveries_queued=deliveries_queued,
        )

        redeliver_event_response_202_data.additional_properties = d
        return redeliver_event_response_202_data

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
