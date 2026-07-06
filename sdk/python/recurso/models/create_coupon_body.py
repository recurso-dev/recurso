from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.create_coupon_body_discount_type import CreateCouponBodyDiscountType
from ..models.create_coupon_body_duration import CreateCouponBodyDuration
from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateCouponBody")


@_attrs_define
class CreateCouponBody:
    """
    Attributes:
        code (str):
        discount_type (CreateCouponBodyDiscountType):
        discount_value (int): Percentage (0-100) when `discount_type` is `percent`, otherwise amount in the lowest
            currency unit.
        duration (CreateCouponBodyDuration):
        duration_months (int | Unset): Required when `duration` is `repeating`.
    """

    code: str
    discount_type: CreateCouponBodyDiscountType
    discount_value: int
    duration: CreateCouponBodyDuration
    duration_months: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        code = self.code

        discount_type = self.discount_type.value

        discount_value = self.discount_value

        duration = self.duration.value

        duration_months = self.duration_months

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "code": code,
                "discount_type": discount_type,
                "discount_value": discount_value,
                "duration": duration,
            }
        )
        if duration_months is not UNSET:
            field_dict["duration_months"] = duration_months

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        discount_type = CreateCouponBodyDiscountType(d.pop("discount_type"))

        discount_value = d.pop("discount_value")

        duration = CreateCouponBodyDuration(d.pop("duration"))

        duration_months = d.pop("duration_months", UNSET)

        create_coupon_body = cls(
            code=code,
            discount_type=discount_type,
            discount_value=discount_value,
            duration=duration,
            duration_months=duration_months,
        )

        create_coupon_body.additional_properties = d
        return create_coupon_body

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
