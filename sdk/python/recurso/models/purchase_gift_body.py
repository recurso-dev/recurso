from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PurchaseGiftBody")


@_attrs_define
class PurchaseGiftBody:
    """
    Attributes:
        buyer_customer_id (UUID):
        plan_id (UUID):
        duration_months (int):
        recipient_email (str | Unset):
    """

    buyer_customer_id: UUID
    plan_id: UUID
    duration_months: int
    recipient_email: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        buyer_customer_id = str(self.buyer_customer_id)

        plan_id = str(self.plan_id)

        duration_months = self.duration_months

        recipient_email = self.recipient_email

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "buyer_customer_id": buyer_customer_id,
                "plan_id": plan_id,
                "duration_months": duration_months,
            }
        )
        if recipient_email is not UNSET:
            field_dict["recipient_email"] = recipient_email

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        buyer_customer_id = UUID(d.pop("buyer_customer_id"))

        plan_id = UUID(d.pop("plan_id"))

        duration_months = d.pop("duration_months")

        recipient_email = d.pop("recipient_email", UNSET)

        purchase_gift_body = cls(
            buyer_customer_id=buyer_customer_id,
            plan_id=plan_id,
            duration_months=duration_months,
            recipient_email=recipient_email,
        )

        purchase_gift_body.additional_properties = d
        return purchase_gift_body

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
