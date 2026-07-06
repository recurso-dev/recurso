from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MRRCurrencyBreakdown")


@_attrs_define
class MRRCurrencyBreakdown:
    """
    Attributes:
        currency (str | Unset):
        amount (int | Unset):
        converted_amount (int | Unset):
        rate (float | Unset):
        subscriptions (int | Unset):
        error (str | Unset): Present only when this currency could not be converted.
    """

    currency: str | Unset = UNSET
    amount: int | Unset = UNSET
    converted_amount: int | Unset = UNSET
    rate: float | Unset = UNSET
    subscriptions: int | Unset = UNSET
    error: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        amount = self.amount

        converted_amount = self.converted_amount

        rate = self.rate

        subscriptions = self.subscriptions

        error = self.error

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if currency is not UNSET:
            field_dict["currency"] = currency
        if amount is not UNSET:
            field_dict["amount"] = amount
        if converted_amount is not UNSET:
            field_dict["converted_amount"] = converted_amount
        if rate is not UNSET:
            field_dict["rate"] = rate
        if subscriptions is not UNSET:
            field_dict["subscriptions"] = subscriptions
        if error is not UNSET:
            field_dict["error"] = error

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency", UNSET)

        amount = d.pop("amount", UNSET)

        converted_amount = d.pop("converted_amount", UNSET)

        rate = d.pop("rate", UNSET)

        subscriptions = d.pop("subscriptions", UNSET)

        error = d.pop("error", UNSET)

        mrr_currency_breakdown = cls(
            currency=currency,
            amount=amount,
            converted_amount=converted_amount,
            rate=rate,
            subscriptions=subscriptions,
            error=error,
        )

        mrr_currency_breakdown.additional_properties = d
        return mrr_currency_breakdown

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
