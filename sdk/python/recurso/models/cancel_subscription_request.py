from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.cancel_subscription_request_reason import CancelSubscriptionRequestReason
from ..types import UNSET, Unset

T = TypeVar("T", bound="CancelSubscriptionRequest")


@_attrs_define
class CancelSubscriptionRequest:
    """
    Attributes:
        reason (CancelSubscriptionRequestReason):
        cancel_at_period_end (bool | Unset): Defaults to true when `immediately` is false.
        immediately (bool | Unset):
        feedback (str | Unset):
        revoke_consent (bool | Unset): Also revoke the recurring-payment consent (RBI compliance).
    """

    reason: CancelSubscriptionRequestReason
    cancel_at_period_end: bool | Unset = UNSET
    immediately: bool | Unset = UNSET
    feedback: str | Unset = UNSET
    revoke_consent: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        reason = self.reason.value

        cancel_at_period_end = self.cancel_at_period_end

        immediately = self.immediately

        feedback = self.feedback

        revoke_consent = self.revoke_consent

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "reason": reason,
            }
        )
        if cancel_at_period_end is not UNSET:
            field_dict["cancel_at_period_end"] = cancel_at_period_end
        if immediately is not UNSET:
            field_dict["immediately"] = immediately
        if feedback is not UNSET:
            field_dict["feedback"] = feedback
        if revoke_consent is not UNSET:
            field_dict["revoke_consent"] = revoke_consent

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reason = CancelSubscriptionRequestReason(d.pop("reason"))

        cancel_at_period_end = d.pop("cancel_at_period_end", UNSET)

        immediately = d.pop("immediately", UNSET)

        feedback = d.pop("feedback", UNSET)

        revoke_consent = d.pop("revoke_consent", UNSET)

        cancel_subscription_request = cls(
            reason=reason,
            cancel_at_period_end=cancel_at_period_end,
            immediately=immediately,
            feedback=feedback,
            revoke_consent=revoke_consent,
        )

        cancel_subscription_request.additional_properties = d
        return cancel_subscription_request

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
