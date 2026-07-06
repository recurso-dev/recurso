from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.subscription_status import SubscriptionStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="Subscription")


@_attrs_define
class Subscription:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        plan_id (UUID | Unset):
        status (SubscriptionStatus | Unset):
        current_period_start (datetime.datetime | Unset):
        current_period_end (datetime.datetime | Unset):
        cancel_at_period_end (bool | Unset):
        canceled_at (datetime.datetime | Unset):
        cancellation_reason (str | Unset):
        cancellation_feedback (str | Unset):
        billing_anchor (datetime.datetime | Unset):
        billing_anchor_type (str | Unset):
        billing_anchor_day (int | Unset):
        payment_terms (str | Unset):
        coupon_id (UUID | Unset):
        reference_id (str | Unset):
        mandate_id (UUID | Unset):
        razorpay_subscription_id (str | Unset):
        stripe_subscription_id (str | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    plan_id: UUID | Unset = UNSET
    status: SubscriptionStatus | Unset = UNSET
    current_period_start: datetime.datetime | Unset = UNSET
    current_period_end: datetime.datetime | Unset = UNSET
    cancel_at_period_end: bool | Unset = UNSET
    canceled_at: datetime.datetime | Unset = UNSET
    cancellation_reason: str | Unset = UNSET
    cancellation_feedback: str | Unset = UNSET
    billing_anchor: datetime.datetime | Unset = UNSET
    billing_anchor_type: str | Unset = UNSET
    billing_anchor_day: int | Unset = UNSET
    payment_terms: str | Unset = UNSET
    coupon_id: UUID | Unset = UNSET
    reference_id: str | Unset = UNSET
    mandate_id: UUID | Unset = UNSET
    razorpay_subscription_id: str | Unset = UNSET
    stripe_subscription_id: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        plan_id: str | Unset = UNSET
        if not isinstance(self.plan_id, Unset):
            plan_id = str(self.plan_id)

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        current_period_start: str | Unset = UNSET
        if not isinstance(self.current_period_start, Unset):
            current_period_start = self.current_period_start.isoformat()

        current_period_end: str | Unset = UNSET
        if not isinstance(self.current_period_end, Unset):
            current_period_end = self.current_period_end.isoformat()

        cancel_at_period_end = self.cancel_at_period_end

        canceled_at: str | Unset = UNSET
        if not isinstance(self.canceled_at, Unset):
            canceled_at = self.canceled_at.isoformat()

        cancellation_reason = self.cancellation_reason

        cancellation_feedback = self.cancellation_feedback

        billing_anchor: str | Unset = UNSET
        if not isinstance(self.billing_anchor, Unset):
            billing_anchor = self.billing_anchor.isoformat()

        billing_anchor_type = self.billing_anchor_type

        billing_anchor_day = self.billing_anchor_day

        payment_terms = self.payment_terms

        coupon_id: str | Unset = UNSET
        if not isinstance(self.coupon_id, Unset):
            coupon_id = str(self.coupon_id)

        reference_id = self.reference_id

        mandate_id: str | Unset = UNSET
        if not isinstance(self.mandate_id, Unset):
            mandate_id = str(self.mandate_id)

        razorpay_subscription_id = self.razorpay_subscription_id

        stripe_subscription_id = self.stripe_subscription_id

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if plan_id is not UNSET:
            field_dict["plan_id"] = plan_id
        if status is not UNSET:
            field_dict["status"] = status
        if current_period_start is not UNSET:
            field_dict["current_period_start"] = current_period_start
        if current_period_end is not UNSET:
            field_dict["current_period_end"] = current_period_end
        if cancel_at_period_end is not UNSET:
            field_dict["cancel_at_period_end"] = cancel_at_period_end
        if canceled_at is not UNSET:
            field_dict["canceled_at"] = canceled_at
        if cancellation_reason is not UNSET:
            field_dict["cancellation_reason"] = cancellation_reason
        if cancellation_feedback is not UNSET:
            field_dict["cancellation_feedback"] = cancellation_feedback
        if billing_anchor is not UNSET:
            field_dict["billing_anchor"] = billing_anchor
        if billing_anchor_type is not UNSET:
            field_dict["billing_anchor_type"] = billing_anchor_type
        if billing_anchor_day is not UNSET:
            field_dict["billing_anchor_day"] = billing_anchor_day
        if payment_terms is not UNSET:
            field_dict["payment_terms"] = payment_terms
        if coupon_id is not UNSET:
            field_dict["coupon_id"] = coupon_id
        if reference_id is not UNSET:
            field_dict["reference_id"] = reference_id
        if mandate_id is not UNSET:
            field_dict["mandate_id"] = mandate_id
        if razorpay_subscription_id is not UNSET:
            field_dict["razorpay_subscription_id"] = razorpay_subscription_id
        if stripe_subscription_id is not UNSET:
            field_dict["stripe_subscription_id"] = stripe_subscription_id
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at

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

        _tenant_id = d.pop("tenant_id", UNSET)
        tenant_id: UUID | Unset
        if isinstance(_tenant_id, Unset):
            tenant_id = UNSET
        else:
            tenant_id = UUID(_tenant_id)

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        _plan_id = d.pop("plan_id", UNSET)
        plan_id: UUID | Unset
        if isinstance(_plan_id, Unset):
            plan_id = UNSET
        else:
            plan_id = UUID(_plan_id)

        _status = d.pop("status", UNSET)
        status: SubscriptionStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = SubscriptionStatus(_status)

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

        cancel_at_period_end = d.pop("cancel_at_period_end", UNSET)

        _canceled_at = d.pop("canceled_at", UNSET)
        canceled_at: datetime.datetime | Unset
        if isinstance(_canceled_at, Unset):
            canceled_at = UNSET
        else:
            canceled_at = datetime.datetime.fromisoformat(_canceled_at)

        cancellation_reason = d.pop("cancellation_reason", UNSET)

        cancellation_feedback = d.pop("cancellation_feedback", UNSET)

        _billing_anchor = d.pop("billing_anchor", UNSET)
        billing_anchor: datetime.datetime | Unset
        if isinstance(_billing_anchor, Unset):
            billing_anchor = UNSET
        else:
            billing_anchor = datetime.datetime.fromisoformat(_billing_anchor)

        billing_anchor_type = d.pop("billing_anchor_type", UNSET)

        billing_anchor_day = d.pop("billing_anchor_day", UNSET)

        payment_terms = d.pop("payment_terms", UNSET)

        _coupon_id = d.pop("coupon_id", UNSET)
        coupon_id: UUID | Unset
        if isinstance(_coupon_id, Unset):
            coupon_id = UNSET
        else:
            coupon_id = UUID(_coupon_id)

        reference_id = d.pop("reference_id", UNSET)

        _mandate_id = d.pop("mandate_id", UNSET)
        mandate_id: UUID | Unset
        if isinstance(_mandate_id, Unset):
            mandate_id = UNSET
        else:
            mandate_id = UUID(_mandate_id)

        razorpay_subscription_id = d.pop("razorpay_subscription_id", UNSET)

        stripe_subscription_id = d.pop("stripe_subscription_id", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        _updated_at = d.pop("updated_at", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at, Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)

        subscription = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            plan_id=plan_id,
            status=status,
            current_period_start=current_period_start,
            current_period_end=current_period_end,
            cancel_at_period_end=cancel_at_period_end,
            canceled_at=canceled_at,
            cancellation_reason=cancellation_reason,
            cancellation_feedback=cancellation_feedback,
            billing_anchor=billing_anchor,
            billing_anchor_type=billing_anchor_type,
            billing_anchor_day=billing_anchor_day,
            payment_terms=payment_terms,
            coupon_id=coupon_id,
            reference_id=reference_id,
            mandate_id=mandate_id,
            razorpay_subscription_id=razorpay_subscription_id,
            stripe_subscription_id=stripe_subscription_id,
            created_at=created_at,
            updated_at=updated_at,
        )

        subscription.additional_properties = d
        return subscription

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
