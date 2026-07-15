from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.plan_interval_unit import PlanIntervalUnit
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.price import Price


T = TypeVar("T", bound="Plan")


@_attrs_define
class Plan:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        name (str | Unset):
        code (str | Unset):
        interval_unit (PlanIntervalUnit | Unset):
        interval_count (int | Unset):
        active (bool | Unset):
        hsn_code (str | Unset): HSN/SAC code the plan's invoice lines are taxed at (empty = tenant SAC default).
        created_at (datetime.datetime | Unset):
        prices (list[Price] | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    name: str | Unset = UNSET
    code: str | Unset = UNSET
    interval_unit: PlanIntervalUnit | Unset = UNSET
    interval_count: int | Unset = UNSET
    active: bool | Unset = UNSET
    hsn_code: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    prices: list[Price] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        name = self.name

        code = self.code

        interval_unit: str | Unset = UNSET
        if not isinstance(self.interval_unit, Unset):
            interval_unit = self.interval_unit.value

        interval_count = self.interval_count

        active = self.active

        hsn_code = self.hsn_code

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        prices: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.prices, Unset):
            prices = []
            for prices_item_data in self.prices:
                prices_item = prices_item_data.to_dict()
                prices.append(prices_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if name is not UNSET:
            field_dict["name"] = name
        if code is not UNSET:
            field_dict["code"] = code
        if interval_unit is not UNSET:
            field_dict["interval_unit"] = interval_unit
        if interval_count is not UNSET:
            field_dict["interval_count"] = interval_count
        if active is not UNSET:
            field_dict["active"] = active
        if hsn_code is not UNSET:
            field_dict["hsn_code"] = hsn_code
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if prices is not UNSET:
            field_dict["prices"] = prices

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.price import Price

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

        name = d.pop("name", UNSET)

        code = d.pop("code", UNSET)

        _interval_unit = d.pop("interval_unit", UNSET)
        interval_unit: PlanIntervalUnit | Unset
        if isinstance(_interval_unit, Unset):
            interval_unit = UNSET
        else:
            interval_unit = PlanIntervalUnit(_interval_unit)

        interval_count = d.pop("interval_count", UNSET)

        active = d.pop("active", UNSET)

        hsn_code = d.pop("hsn_code", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        _prices = d.pop("prices", UNSET)
        prices: list[Price] | Unset = UNSET
        if _prices is not UNSET:
            prices = []
            for prices_item_data in _prices:
                prices_item = Price.from_dict(prices_item_data)

                prices.append(prices_item)

        plan = cls(
            id=id,
            tenant_id=tenant_id,
            name=name,
            code=code,
            interval_unit=interval_unit,
            interval_count=interval_count,
            active=active,
            hsn_code=hsn_code,
            created_at=created_at,
            prices=prices,
        )

        plan.additional_properties = d
        return plan

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
