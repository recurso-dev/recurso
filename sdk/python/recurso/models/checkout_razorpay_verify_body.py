from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="CheckoutRazorpayVerifyBody")


@_attrs_define
class CheckoutRazorpayVerifyBody:
    """
    Attributes:
        razorpay_order_id (str):
        razorpay_payment_id (str):
        razorpay_signature (str):
    """

    razorpay_order_id: str
    razorpay_payment_id: str
    razorpay_signature: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        razorpay_order_id = self.razorpay_order_id

        razorpay_payment_id = self.razorpay_payment_id

        razorpay_signature = self.razorpay_signature

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "razorpay_order_id": razorpay_order_id,
                "razorpay_payment_id": razorpay_payment_id,
                "razorpay_signature": razorpay_signature,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        razorpay_order_id = d.pop("razorpay_order_id")

        razorpay_payment_id = d.pop("razorpay_payment_id")

        razorpay_signature = d.pop("razorpay_signature")

        checkout_razorpay_verify_body = cls(
            razorpay_order_id=razorpay_order_id,
            razorpay_payment_id=razorpay_payment_id,
            razorpay_signature=razorpay_signature,
        )

        checkout_razorpay_verify_body.additional_properties = d
        return checkout_razorpay_verify_body

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
