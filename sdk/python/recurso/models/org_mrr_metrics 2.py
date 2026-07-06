from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.currency_mrr import CurrencyMRR


T = TypeVar("T", bound="OrgMRRMetrics")


@_attrs_define
class OrgMRRMetrics:
    """
    Attributes:
        by_currency (list[CurrencyMRR] | Unset):
    """

    by_currency: list[CurrencyMRR] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        by_currency: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.by_currency, Unset):
            by_currency = []
            for by_currency_item_data in self.by_currency:
                by_currency_item = by_currency_item_data.to_dict()
                by_currency.append(by_currency_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if by_currency is not UNSET:
            field_dict["by_currency"] = by_currency

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.currency_mrr import CurrencyMRR

        d = dict(src_dict)
        _by_currency = d.pop("by_currency", UNSET)
        by_currency: list[CurrencyMRR] | Unset = UNSET
        if _by_currency is not UNSET:
            by_currency = []
            for by_currency_item_data in _by_currency:
                by_currency_item = CurrencyMRR.from_dict(by_currency_item_data)

                by_currency.append(by_currency_item)

        org_mrr_metrics = cls(
            by_currency=by_currency,
        )

        org_mrr_metrics.additional_properties = d
        return org_mrr_metrics

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
