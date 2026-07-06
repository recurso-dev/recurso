from __future__ import annotations

from collections.abc import Mapping
from typing import Any, Literal, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RecordUsageEventResponse201")


@_attrs_define
class RecordUsageEventResponse201:
    """
    Attributes:
        status (Literal['recorded'] | Unset):
        event_id (UUID | Unset):
    """

    status: Literal["recorded"] | Unset = UNSET
    event_id: UUID | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status = self.status

        event_id: str | Unset = UNSET
        if not isinstance(self.event_id, Unset):
            event_id = str(self.event_id)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if status is not UNSET:
            field_dict["status"] = status
        if event_id is not UNSET:
            field_dict["event_id"] = event_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = cast(Literal["recorded"] | Unset, d.pop("status", UNSET))
        if status != "recorded" and not isinstance(status, Unset):
            raise ValueError(f"status must match const 'recorded', got '{status}'")

        _event_id = d.pop("event_id", UNSET)
        event_id: UUID | Unset
        if isinstance(_event_id, Unset):
            event_id = UNSET
        else:
            event_id = UUID(_event_id)

        record_usage_event_response_201 = cls(
            status=status,
            event_id=event_id,
        )

        record_usage_event_response_201.additional_properties = d
        return record_usage_event_response_201

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
