from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.create_plan_request_interval_unit import CreatePlanRequestIntervalUnit
from ..types import UNSET, Unset

T = TypeVar("T", bound="CreatePlanRequest")


@_attrs_define
class CreatePlanRequest:
    """
    Attributes:
        name (str):
        code (str): Unique plan code (e.g. `gold-monthly`).
        interval_unit (CreatePlanRequestIntervalUnit):
        interval_count (int):
        amount (int): Price in the lowest currency unit (e.g. cents/paise).
        currency (str): ISO 4217 code.
        hsn_code (str | Unset): Optional HSN/SAC code for this plan. Each invoice line for the plan is taxed at this
            code's GST rate. Empty falls back to the tenant SAC (then the 998314 default).
    """

    name: str
    code: str
    interval_unit: CreatePlanRequestIntervalUnit
    interval_count: int
    amount: int
    currency: str
    hsn_code: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        code = self.code

        interval_unit = self.interval_unit.value

        interval_count = self.interval_count

        amount = self.amount

        currency = self.currency

        hsn_code = self.hsn_code

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "code": code,
                "interval_unit": interval_unit,
                "interval_count": interval_count,
                "amount": amount,
                "currency": currency,
            }
        )
        if hsn_code is not UNSET:
            field_dict["hsn_code"] = hsn_code

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        code = d.pop("code")

        interval_unit = CreatePlanRequestIntervalUnit(d.pop("interval_unit"))

        interval_count = d.pop("interval_count")

        amount = d.pop("amount")

        currency = d.pop("currency")

        hsn_code = d.pop("hsn_code", UNSET)

        create_plan_request = cls(
            name=name,
            code=code,
            interval_unit=interval_unit,
            interval_count=interval_count,
            amount=amount,
            currency=currency,
            hsn_code=hsn_code,
        )

        create_plan_request.additional_properties = d
        return create_plan_request

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
