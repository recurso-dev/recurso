from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.coupon_discount_type import CouponDiscountType
from ..models.coupon_duration import CouponDuration
from ..types import UNSET, Unset

T = TypeVar("T", bound="Coupon")


@_attrs_define
class Coupon:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        code (str | Unset):
        discount_type (CouponDiscountType | Unset):
        discount_value (int | Unset):
        duration (CouponDuration | Unset):
        duration_months (int | None | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    code: str | Unset = UNSET
    discount_type: CouponDiscountType | Unset = UNSET
    discount_value: int | Unset = UNSET
    duration: CouponDuration | Unset = UNSET
    duration_months: int | None | Unset = UNSET
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

        code = self.code

        discount_type: str | Unset = UNSET
        if not isinstance(self.discount_type, Unset):
            discount_type = self.discount_type.value

        discount_value = self.discount_value

        duration: str | Unset = UNSET
        if not isinstance(self.duration, Unset):
            duration = self.duration.value

        duration_months: int | None | Unset
        if isinstance(self.duration_months, Unset):
            duration_months = UNSET
        else:
            duration_months = self.duration_months

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
        if code is not UNSET:
            field_dict["code"] = code
        if discount_type is not UNSET:
            field_dict["discount_type"] = discount_type
        if discount_value is not UNSET:
            field_dict["discount_value"] = discount_value
        if duration is not UNSET:
            field_dict["duration"] = duration
        if duration_months is not UNSET:
            field_dict["duration_months"] = duration_months
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

        code = d.pop("code", UNSET)

        _discount_type = d.pop("discount_type", UNSET)
        discount_type: CouponDiscountType | Unset
        if isinstance(_discount_type, Unset):
            discount_type = UNSET
        else:
            discount_type = CouponDiscountType(_discount_type)

        discount_value = d.pop("discount_value", UNSET)

        _duration = d.pop("duration", UNSET)
        duration: CouponDuration | Unset
        if isinstance(_duration, Unset):
            duration = UNSET
        else:
            duration = CouponDuration(_duration)

        def _parse_duration_months(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        duration_months = _parse_duration_months(d.pop("duration_months", UNSET))

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

        coupon = cls(
            id=id,
            tenant_id=tenant_id,
            code=code,
            discount_type=discount_type,
            discount_value=discount_value,
            duration=duration,
            duration_months=duration_months,
            created_at=created_at,
            updated_at=updated_at,
        )

        coupon.additional_properties = d
        return coupon

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
