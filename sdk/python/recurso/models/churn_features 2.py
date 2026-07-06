from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ChurnFeatures")


@_attrs_define
class ChurnFeatures:
    """
    Attributes:
        days_since_signup (int | Unset):
        total_invoices (int | Unset):
        failed_invoices_90d (int | Unset):
        payment_failure_rate (float | Unset):
        avg_days_to_pay (float | Unset):
        plan_downgrades (int | Unset):
        months_active (int | Unset):
        current_mrr (int | Unset):
        usage_trend (float | Unset):
    """

    days_since_signup: int | Unset = UNSET
    total_invoices: int | Unset = UNSET
    failed_invoices_90d: int | Unset = UNSET
    payment_failure_rate: float | Unset = UNSET
    avg_days_to_pay: float | Unset = UNSET
    plan_downgrades: int | Unset = UNSET
    months_active: int | Unset = UNSET
    current_mrr: int | Unset = UNSET
    usage_trend: float | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        days_since_signup = self.days_since_signup

        total_invoices = self.total_invoices

        failed_invoices_90d = self.failed_invoices_90d

        payment_failure_rate = self.payment_failure_rate

        avg_days_to_pay = self.avg_days_to_pay

        plan_downgrades = self.plan_downgrades

        months_active = self.months_active

        current_mrr = self.current_mrr

        usage_trend = self.usage_trend

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if days_since_signup is not UNSET:
            field_dict["days_since_signup"] = days_since_signup
        if total_invoices is not UNSET:
            field_dict["total_invoices"] = total_invoices
        if failed_invoices_90d is not UNSET:
            field_dict["failed_invoices_90d"] = failed_invoices_90d
        if payment_failure_rate is not UNSET:
            field_dict["payment_failure_rate"] = payment_failure_rate
        if avg_days_to_pay is not UNSET:
            field_dict["avg_days_to_pay"] = avg_days_to_pay
        if plan_downgrades is not UNSET:
            field_dict["plan_downgrades"] = plan_downgrades
        if months_active is not UNSET:
            field_dict["months_active"] = months_active
        if current_mrr is not UNSET:
            field_dict["current_mrr"] = current_mrr
        if usage_trend is not UNSET:
            field_dict["usage_trend"] = usage_trend

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        days_since_signup = d.pop("days_since_signup", UNSET)

        total_invoices = d.pop("total_invoices", UNSET)

        failed_invoices_90d = d.pop("failed_invoices_90d", UNSET)

        payment_failure_rate = d.pop("payment_failure_rate", UNSET)

        avg_days_to_pay = d.pop("avg_days_to_pay", UNSET)

        plan_downgrades = d.pop("plan_downgrades", UNSET)

        months_active = d.pop("months_active", UNSET)

        current_mrr = d.pop("current_mrr", UNSET)

        usage_trend = d.pop("usage_trend", UNSET)

        churn_features = cls(
            days_since_signup=days_since_signup,
            total_invoices=total_invoices,
            failed_invoices_90d=failed_invoices_90d,
            payment_failure_rate=payment_failure_rate,
            avg_days_to_pay=avg_days_to_pay,
            plan_downgrades=plan_downgrades,
            months_active=months_active,
            current_mrr=current_mrr,
            usage_trend=usage_trend,
        )

        churn_features.additional_properties = d
        return churn_features

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
