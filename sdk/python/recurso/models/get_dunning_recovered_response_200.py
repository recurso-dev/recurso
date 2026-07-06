from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.get_dunning_recovered_response_200_monthly_item import GetDunningRecoveredResponse200MonthlyItem
    from ..models.get_dunning_recovered_response_200_recovered_amount_total import (
        GetDunningRecoveredResponse200RecoveredAmountTotal,
    )


T = TypeVar("T", bound="GetDunningRecoveredResponse200")


@_attrs_define
class GetDunningRecoveredResponse200:
    """
    Attributes:
        recovered_amount_total (GetDunningRecoveredResponse200RecoveredAmountTotal | Unset): Total recovered, in minor
            units, keyed by currency.
        recovered_count (int | Unset):
        avg_attempts (float | Unset):
        avg_days_to_recover (float | Unset):
        monthly (list[GetDunningRecoveredResponse200MonthlyItem] | Unset):
    """

    recovered_amount_total: GetDunningRecoveredResponse200RecoveredAmountTotal | Unset = UNSET
    recovered_count: int | Unset = UNSET
    avg_attempts: float | Unset = UNSET
    avg_days_to_recover: float | Unset = UNSET
    monthly: list[GetDunningRecoveredResponse200MonthlyItem] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        recovered_amount_total: dict[str, Any] | Unset = UNSET
        if not isinstance(self.recovered_amount_total, Unset):
            recovered_amount_total = self.recovered_amount_total.to_dict()

        recovered_count = self.recovered_count

        avg_attempts = self.avg_attempts

        avg_days_to_recover = self.avg_days_to_recover

        monthly: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.monthly, Unset):
            monthly = []
            for monthly_item_data in self.monthly:
                monthly_item = monthly_item_data.to_dict()
                monthly.append(monthly_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if recovered_amount_total is not UNSET:
            field_dict["recovered_amount_total"] = recovered_amount_total
        if recovered_count is not UNSET:
            field_dict["recovered_count"] = recovered_count
        if avg_attempts is not UNSET:
            field_dict["avg_attempts"] = avg_attempts
        if avg_days_to_recover is not UNSET:
            field_dict["avg_days_to_recover"] = avg_days_to_recover
        if monthly is not UNSET:
            field_dict["monthly"] = monthly

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.get_dunning_recovered_response_200_monthly_item import GetDunningRecoveredResponse200MonthlyItem
        from ..models.get_dunning_recovered_response_200_recovered_amount_total import (
            GetDunningRecoveredResponse200RecoveredAmountTotal,
        )

        d = dict(src_dict)
        _recovered_amount_total = d.pop("recovered_amount_total", UNSET)
        recovered_amount_total: GetDunningRecoveredResponse200RecoveredAmountTotal | Unset
        if isinstance(_recovered_amount_total, Unset):
            recovered_amount_total = UNSET
        else:
            recovered_amount_total = GetDunningRecoveredResponse200RecoveredAmountTotal.from_dict(
                _recovered_amount_total
            )

        recovered_count = d.pop("recovered_count", UNSET)

        avg_attempts = d.pop("avg_attempts", UNSET)

        avg_days_to_recover = d.pop("avg_days_to_recover", UNSET)

        _monthly = d.pop("monthly", UNSET)
        monthly: list[GetDunningRecoveredResponse200MonthlyItem] | Unset = UNSET
        if _monthly is not UNSET:
            monthly = []
            for monthly_item_data in _monthly:
                monthly_item = GetDunningRecoveredResponse200MonthlyItem.from_dict(monthly_item_data)

                monthly.append(monthly_item)

        get_dunning_recovered_response_200 = cls(
            recovered_amount_total=recovered_amount_total,
            recovered_count=recovered_count,
            avg_attempts=avg_attempts,
            avg_days_to_recover=avg_days_to_recover,
            monthly=monthly,
        )

        get_dunning_recovered_response_200.additional_properties = d
        return get_dunning_recovered_response_200

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
