from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CancelSubscriptionResponse")


@_attrs_define
class CancelSubscriptionResponse:
    """
    Attributes:
        id (UUID | Unset):
        status (str | Unset):
        cancel_at_period_end (bool | Unset):
        cancelled_at (datetime.datetime | Unset):
        current_period_end (datetime.datetime | Unset):
        cancellation_reason (str | Unset):
        message (str | Unset):
    """

    id: UUID | Unset = UNSET
    status: str | Unset = UNSET
    cancel_at_period_end: bool | Unset = UNSET
    cancelled_at: datetime.datetime | Unset = UNSET
    current_period_end: datetime.datetime | Unset = UNSET
    cancellation_reason: str | Unset = UNSET
    message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        status = self.status

        cancel_at_period_end = self.cancel_at_period_end

        cancelled_at: str | Unset = UNSET
        if not isinstance(self.cancelled_at, Unset):
            cancelled_at = self.cancelled_at.isoformat()

        current_period_end: str | Unset = UNSET
        if not isinstance(self.current_period_end, Unset):
            current_period_end = self.current_period_end.isoformat()

        cancellation_reason = self.cancellation_reason

        message = self.message

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if status is not UNSET:
            field_dict["status"] = status
        if cancel_at_period_end is not UNSET:
            field_dict["cancel_at_period_end"] = cancel_at_period_end
        if cancelled_at is not UNSET:
            field_dict["cancelled_at"] = cancelled_at
        if current_period_end is not UNSET:
            field_dict["current_period_end"] = current_period_end
        if cancellation_reason is not UNSET:
            field_dict["cancellation_reason"] = cancellation_reason
        if message is not UNSET:
            field_dict["message"] = message

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

        status = d.pop("status", UNSET)

        cancel_at_period_end = d.pop("cancel_at_period_end", UNSET)

        _cancelled_at = d.pop("cancelled_at", UNSET)
        cancelled_at: datetime.datetime | Unset
        if isinstance(_cancelled_at, Unset):
            cancelled_at = UNSET
        else:
            cancelled_at = datetime.datetime.fromisoformat(_cancelled_at)

        _current_period_end = d.pop("current_period_end", UNSET)
        current_period_end: datetime.datetime | Unset
        if isinstance(_current_period_end, Unset):
            current_period_end = UNSET
        else:
            current_period_end = datetime.datetime.fromisoformat(_current_period_end)

        cancellation_reason = d.pop("cancellation_reason", UNSET)

        message = d.pop("message", UNSET)

        cancel_subscription_response = cls(
            id=id,
            status=status,
            cancel_at_period_end=cancel_at_period_end,
            cancelled_at=cancelled_at,
            current_period_end=current_period_end,
            cancellation_reason=cancellation_reason,
            message=message,
        )

        cancel_subscription_response.additional_properties = d
        return cancel_subscription_response

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
