from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.handle_stripe_webhook_response_200_status import HandleStripeWebhookResponse200Status
from ..types import UNSET, Unset

T = TypeVar("T", bound="HandleStripeWebhookResponse200")


@_attrs_define
class HandleStripeWebhookResponse200:
    """
    Attributes:
        status (HandleStripeWebhookResponse200Status | Unset):
        reason (str | Unset): Present when the event was ignored.
    """

    status: HandleStripeWebhookResponse200Status | Unset = UNSET
    reason: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        reason = self.reason

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if status is not UNSET:
            field_dict["status"] = status
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _status = d.pop("status", UNSET)
        status: HandleStripeWebhookResponse200Status | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = HandleStripeWebhookResponse200Status(_status)

        reason = d.pop("reason", UNSET)

        handle_stripe_webhook_response_200 = cls(
            status=status,
            reason=reason,
        )

        handle_stripe_webhook_response_200.additional_properties = d
        return handle_stripe_webhook_response_200

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
