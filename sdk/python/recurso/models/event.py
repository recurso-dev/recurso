from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_data import EventData


T = TypeVar("T", bound="Event")


@_attrs_define
class Event:
    """
    Attributes:
        id (UUID | Unset):
        type_ (str | Unset):
        object_type (str | Unset):
        object_id (UUID | Unset):
        data (EventData | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    type_: str | Unset = UNSET
    object_type: str | Unset = UNSET
    object_id: UUID | Unset = UNSET
    data: EventData | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        type_ = self.type_

        object_type = self.object_type

        object_id: str | Unset = UNSET
        if not isinstance(self.object_id, Unset):
            object_id = str(self.object_id)

        data: dict[str, Any] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = self.data.to_dict()

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if type_ is not UNSET:
            field_dict["type"] = type_
        if object_type is not UNSET:
            field_dict["object_type"] = object_type
        if object_id is not UNSET:
            field_dict["object_id"] = object_id
        if data is not UNSET:
            field_dict["data"] = data
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.event_data import EventData

        d = dict(src_dict)
        _id = d.pop("id", UNSET)
        id: UUID | Unset
        if isinstance(_id, Unset):
            id = UNSET
        else:
            id = UUID(_id)

        type_ = d.pop("type", UNSET)

        object_type = d.pop("object_type", UNSET)

        _object_id = d.pop("object_id", UNSET)
        object_id: UUID | Unset
        if isinstance(_object_id, Unset):
            object_id = UNSET
        else:
            object_id = UUID(_object_id)

        _data = d.pop("data", UNSET)
        data: EventData | Unset
        if isinstance(_data, Unset):
            data = UNSET
        else:
            data = EventData.from_dict(_data)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        event = cls(
            id=id,
            type_=type_,
            object_type=object_type,
            object_id=object_id,
            data=data,
            created_at=created_at,
        )

        event.additional_properties = d
        return event

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
