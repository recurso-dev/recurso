from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="GetDunningRecoveredResponse200MonthlyItem")


@_attrs_define
class GetDunningRecoveredResponse200MonthlyItem:
    """
    Attributes:
        month (str | Unset):
        currency (str | Unset):
        amount (int | Unset):
        count (int | Unset):
    """

    month: str | Unset = UNSET
    currency: str | Unset = UNSET
    amount: int | Unset = UNSET
    count: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        month = self.month

        currency = self.currency

        amount = self.amount

        count = self.count

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if month is not UNSET:
            field_dict["month"] = month
        if currency is not UNSET:
            field_dict["currency"] = currency
        if amount is not UNSET:
            field_dict["amount"] = amount
        if count is not UNSET:
            field_dict["count"] = count

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        month = d.pop("month", UNSET)

        currency = d.pop("currency", UNSET)

        amount = d.pop("amount", UNSET)

        count = d.pop("count", UNSET)

        get_dunning_recovered_response_200_monthly_item = cls(
            month=month,
            currency=currency,
            amount=amount,
            count=count,
        )

        get_dunning_recovered_response_200_monthly_item.additional_properties = d
        return get_dunning_recovered_response_200_monthly_item

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
