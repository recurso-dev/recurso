from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.create_subscription_request_billing_anchor_type import CreateSubscriptionRequestBillingAnchorType
from ..models.create_subscription_request_payment_terms import CreateSubscriptionRequestPaymentTerms
from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateSubscriptionRequest")


@_attrs_define
class CreateSubscriptionRequest:
    """
    Attributes:
        customer_id (UUID):
        plan_id (UUID):
        coupon_code (str | Unset):
        start_date (datetime.datetime | Unset):
        billing_anchor_type (CreateSubscriptionRequestBillingAnchorType | Unset):
        payment_terms (CreateSubscriptionRequestPaymentTerms | Unset):
    """

    customer_id: UUID
    plan_id: UUID
    coupon_code: str | Unset = UNSET
    start_date: datetime.datetime | Unset = UNSET
    billing_anchor_type: CreateSubscriptionRequestBillingAnchorType | Unset = UNSET
    payment_terms: CreateSubscriptionRequestPaymentTerms | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        customer_id = str(self.customer_id)

        plan_id = str(self.plan_id)

        coupon_code = self.coupon_code

        start_date: str | Unset = UNSET
        if not isinstance(self.start_date, Unset):
            start_date = self.start_date.isoformat()

        billing_anchor_type: str | Unset = UNSET
        if not isinstance(self.billing_anchor_type, Unset):
            billing_anchor_type = self.billing_anchor_type.value

        payment_terms: str | Unset = UNSET
        if not isinstance(self.payment_terms, Unset):
            payment_terms = self.payment_terms.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "customer_id": customer_id,
                "plan_id": plan_id,
            }
        )
        if coupon_code is not UNSET:
            field_dict["coupon_code"] = coupon_code
        if start_date is not UNSET:
            field_dict["start_date"] = start_date
        if billing_anchor_type is not UNSET:
            field_dict["billing_anchor_type"] = billing_anchor_type
        if payment_terms is not UNSET:
            field_dict["payment_terms"] = payment_terms

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        customer_id = UUID(d.pop("customer_id"))

        plan_id = UUID(d.pop("plan_id"))

        coupon_code = d.pop("coupon_code", UNSET)

        _start_date = d.pop("start_date", UNSET)
        start_date: datetime.datetime | Unset
        if isinstance(_start_date, Unset):
            start_date = UNSET
        else:
            start_date = datetime.datetime.fromisoformat(_start_date)

        _billing_anchor_type = d.pop("billing_anchor_type", UNSET)
        billing_anchor_type: CreateSubscriptionRequestBillingAnchorType | Unset
        if isinstance(_billing_anchor_type, Unset):
            billing_anchor_type = UNSET
        else:
            billing_anchor_type = CreateSubscriptionRequestBillingAnchorType(_billing_anchor_type)

        _payment_terms = d.pop("payment_terms", UNSET)
        payment_terms: CreateSubscriptionRequestPaymentTerms | Unset
        if isinstance(_payment_terms, Unset):
            payment_terms = UNSET
        else:
            payment_terms = CreateSubscriptionRequestPaymentTerms(_payment_terms)

        create_subscription_request = cls(
            customer_id=customer_id,
            plan_id=plan_id,
            coupon_code=coupon_code,
            start_date=start_date,
            billing_anchor_type=billing_anchor_type,
            payment_terms=payment_terms,
        )

        create_subscription_request.additional_properties = d
        return create_subscription_request

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
