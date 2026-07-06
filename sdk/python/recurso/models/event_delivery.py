from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.event_delivery_status import EventDeliveryStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="EventDelivery")


@_attrs_define
class EventDelivery:
    """
    Attributes:
        id (UUID | Unset):
        event_id (UUID | Unset):
        webhook_endpoint_id (UUID | Unset):
        endpoint_url (str | Unset):
        status (EventDeliveryStatus | Unset): Derived delivery state. `pending` deliveries are still being retried by
            the worker; `failed` means retries were exhausted.
        attempts (int | Unset): Number of delivery attempts made so far.
        last_status_code (int | Unset): HTTP status of the most recent attempt. Omitted when no response was received
            (transport error or no attempt yet).
        last_error (str | Unset): Failure reason of the most recent attempt (transport error or "HTTP <code>: <body>").
            Omitted for succeeded deliveries.
        next_retry_at (datetime.datetime | Unset): When the worker will retry next. Omitted once terminal or before the
            first attempt.
        delivered_at (datetime.datetime | Unset): When the delivery reached a terminal state.
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    event_id: UUID | Unset = UNSET
    webhook_endpoint_id: UUID | Unset = UNSET
    endpoint_url: str | Unset = UNSET
    status: EventDeliveryStatus | Unset = UNSET
    attempts: int | Unset = UNSET
    last_status_code: int | Unset = UNSET
    last_error: str | Unset = UNSET
    next_retry_at: datetime.datetime | Unset = UNSET
    delivered_at: datetime.datetime | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        event_id: str | Unset = UNSET
        if not isinstance(self.event_id, Unset):
            event_id = str(self.event_id)

        webhook_endpoint_id: str | Unset = UNSET
        if not isinstance(self.webhook_endpoint_id, Unset):
            webhook_endpoint_id = str(self.webhook_endpoint_id)

        endpoint_url = self.endpoint_url

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        attempts = self.attempts

        last_status_code = self.last_status_code

        last_error = self.last_error

        next_retry_at: str | Unset = UNSET
        if not isinstance(self.next_retry_at, Unset):
            next_retry_at = self.next_retry_at.isoformat()

        delivered_at: str | Unset = UNSET
        if not isinstance(self.delivered_at, Unset):
            delivered_at = self.delivered_at.isoformat()

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if event_id is not UNSET:
            field_dict["event_id"] = event_id
        if webhook_endpoint_id is not UNSET:
            field_dict["webhook_endpoint_id"] = webhook_endpoint_id
        if endpoint_url is not UNSET:
            field_dict["endpoint_url"] = endpoint_url
        if status is not UNSET:
            field_dict["status"] = status
        if attempts is not UNSET:
            field_dict["attempts"] = attempts
        if last_status_code is not UNSET:
            field_dict["last_status_code"] = last_status_code
        if last_error is not UNSET:
            field_dict["last_error"] = last_error
        if next_retry_at is not UNSET:
            field_dict["next_retry_at"] = next_retry_at
        if delivered_at is not UNSET:
            field_dict["delivered_at"] = delivered_at
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

        _event_id = d.pop("event_id", UNSET)
        event_id: UUID | Unset
        if isinstance(_event_id, Unset):
            event_id = UNSET
        else:
            event_id = UUID(_event_id)

        _webhook_endpoint_id = d.pop("webhook_endpoint_id", UNSET)
        webhook_endpoint_id: UUID | Unset
        if isinstance(_webhook_endpoint_id, Unset):
            webhook_endpoint_id = UNSET
        else:
            webhook_endpoint_id = UUID(_webhook_endpoint_id)

        endpoint_url = d.pop("endpoint_url", UNSET)

        _status = d.pop("status", UNSET)
        status: EventDeliveryStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = EventDeliveryStatus(_status)

        attempts = d.pop("attempts", UNSET)

        last_status_code = d.pop("last_status_code", UNSET)

        last_error = d.pop("last_error", UNSET)

        _next_retry_at = d.pop("next_retry_at", UNSET)
        next_retry_at: datetime.datetime | Unset
        if isinstance(_next_retry_at, Unset):
            next_retry_at = UNSET
        else:
            next_retry_at = datetime.datetime.fromisoformat(_next_retry_at)

        _delivered_at = d.pop("delivered_at", UNSET)
        delivered_at: datetime.datetime | Unset
        if isinstance(_delivered_at, Unset):
            delivered_at = UNSET
        else:
            delivered_at = datetime.datetime.fromisoformat(_delivered_at)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        event_delivery = cls(
            id=id,
            event_id=event_id,
            webhook_endpoint_id=webhook_endpoint_id,
            endpoint_url=endpoint_url,
            status=status,
            attempts=attempts,
            last_status_code=last_status_code,
            last_error=last_error,
            next_retry_at=next_retry_at,
            delivered_at=delivered_at,
            created_at=created_at,
        )

        event_delivery.additional_properties = d
        return event_delivery

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
