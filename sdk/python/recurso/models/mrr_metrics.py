from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.fx_snapshot import FXSnapshot
    from ..models.mrr_currency_breakdown import MRRCurrencyBreakdown


T = TypeVar("T", bound="MRRMetrics")


@_attrs_define
class MRRMetrics:
    """
    Attributes:
        currency (str | Unset): The reporting currency.
        amount (int | Unset): FX-normalized MRR total in the reporting currency, minor units.
        mrr (int | Unset): Alias of the normalized total.
        normalized_mrr (int | Unset):
        reporting_currency (str | Unset):
        breakdown (list[MRRCurrencyBreakdown] | Unset):
        fx (FXSnapshot | Unset):
    """

    currency: str | Unset = UNSET
    amount: int | Unset = UNSET
    mrr: int | Unset = UNSET
    normalized_mrr: int | Unset = UNSET
    reporting_currency: str | Unset = UNSET
    breakdown: list[MRRCurrencyBreakdown] | Unset = UNSET
    fx: FXSnapshot | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        amount = self.amount

        mrr = self.mrr

        normalized_mrr = self.normalized_mrr

        reporting_currency = self.reporting_currency

        breakdown: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.breakdown, Unset):
            breakdown = []
            for breakdown_item_data in self.breakdown:
                breakdown_item = breakdown_item_data.to_dict()
                breakdown.append(breakdown_item)

        fx: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fx, Unset):
            fx = self.fx.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if currency is not UNSET:
            field_dict["currency"] = currency
        if amount is not UNSET:
            field_dict["amount"] = amount
        if mrr is not UNSET:
            field_dict["mrr"] = mrr
        if normalized_mrr is not UNSET:
            field_dict["normalized_mrr"] = normalized_mrr
        if reporting_currency is not UNSET:
            field_dict["reporting_currency"] = reporting_currency
        if breakdown is not UNSET:
            field_dict["breakdown"] = breakdown
        if fx is not UNSET:
            field_dict["fx"] = fx

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.fx_snapshot import FXSnapshot
        from ..models.mrr_currency_breakdown import MRRCurrencyBreakdown

        d = dict(src_dict)
        currency = d.pop("currency", UNSET)

        amount = d.pop("amount", UNSET)

        mrr = d.pop("mrr", UNSET)

        normalized_mrr = d.pop("normalized_mrr", UNSET)

        reporting_currency = d.pop("reporting_currency", UNSET)

        _breakdown = d.pop("breakdown", UNSET)
        breakdown: list[MRRCurrencyBreakdown] | Unset = UNSET
        if _breakdown is not UNSET:
            breakdown = []
            for breakdown_item_data in _breakdown:
                breakdown_item = MRRCurrencyBreakdown.from_dict(breakdown_item_data)

                breakdown.append(breakdown_item)

        _fx = d.pop("fx", UNSET)
        fx: FXSnapshot | Unset
        if isinstance(_fx, Unset):
            fx = UNSET
        else:
            fx = FXSnapshot.from_dict(_fx)

        mrr_metrics = cls(
            currency=currency,
            amount=amount,
            mrr=mrr,
            normalized_mrr=normalized_mrr,
            reporting_currency=reporting_currency,
            breakdown=breakdown,
            fx=fx,
        )

        mrr_metrics.additional_properties = d
        return mrr_metrics

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
