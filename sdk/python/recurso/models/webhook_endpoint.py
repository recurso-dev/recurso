from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WebhookEndpoint")


@_attrs_define
class WebhookEndpoint:
    """
    Attributes:
        id (UUID | Unset):
        url (str | Unset):
        secret (str | Unset): HMAC signing secret. Returned only on creation.
        events (list[str] | Unset):
        status (str | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    url: str | Unset = UNSET
    secret: str | Unset = UNSET
    events: list[str] | Unset = UNSET
    status: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        url = self.url

        secret = self.secret

        events: list[str] | Unset = UNSET
        if not isinstance(self.events, Unset):
            events = self.events

        status = self.status

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if url is not UNSET:
            field_dict["url"] = url
        if secret is not UNSET:
            field_dict["secret"] = secret
        if events is not UNSET:
            field_dict["events"] = events
        if status is not UNSET:
            field_dict["status"] = status
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _id = d.pop("id", UNSET)
        id: UUID | Unset
        if isinstance(_id, Unset):
            id = UNSET
        else:
            id = UUID(_id)

        url = d.pop("url", UNSET)

        secret = d.pop("secret", UNSET)

        events = cast(list[str], d.pop("events", UNSET))

        status = d.pop("status", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        webhook_endpoint = cls(
            id=id,
            url=url,
            secret=secret,
            events=events,
            status=status,
            created_at=created_at,
        )

        webhook_endpoint.additional_properties = d
        return webhook_endpoint

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
