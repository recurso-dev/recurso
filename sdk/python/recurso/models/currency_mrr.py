from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.tenant_mrr import TenantMRR


T = TypeVar("T", bound="CurrencyMRR")


@_attrs_define
class CurrencyMRR:
    """
    Attributes:
        total_mrr (int | Unset):
        currency (str | Unset):
        converted_mrr (int | Unset):
        rate (float | Unset):
        fx_error (str | Unset): Present only when this currency could not be converted.
        by_tenant (list[TenantMRR] | Unset):
    """

    total_mrr: int | Unset = UNSET
    currency: str | Unset = UNSET
    converted_mrr: int | Unset = UNSET
    rate: float | Unset = UNSET
    fx_error: str | Unset = UNSET
    by_tenant: list[TenantMRR] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        total_mrr = self.total_mrr

        currency = self.currency

        converted_mrr = self.converted_mrr

        rate = self.rate

        fx_error = self.fx_error

        by_tenant: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.by_tenant, Unset):
            by_tenant = []
            for by_tenant_item_data in self.by_tenant:
                by_tenant_item = by_tenant_item_data.to_dict()
                by_tenant.append(by_tenant_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if total_mrr is not UNSET:
            field_dict["total_mrr"] = total_mrr
        if currency is not UNSET:
            field_dict["currency"] = currency
        if converted_mrr is not UNSET:
            field_dict["converted_mrr"] = converted_mrr
        if rate is not UNSET:
            field_dict["rate"] = rate
        if fx_error is not UNSET:
            field_dict["fx_error"] = fx_error
        if by_tenant is not UNSET:
            field_dict["by_tenant"] = by_tenant

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tenant_mrr import TenantMRR

        d = dict(src_dict)
        total_mrr = d.pop("total_mrr", UNSET)

        currency = d.pop("currency", UNSET)

        converted_mrr = d.pop("converted_mrr", UNSET)

        rate = d.pop("rate", UNSET)

        fx_error = d.pop("fx_error", UNSET)

        _by_tenant = d.pop("by_tenant", UNSET)
        by_tenant: list[TenantMRR] | Unset = UNSET
        if _by_tenant is not UNSET:
            by_tenant = []
            for by_tenant_item_data in _by_tenant:
                by_tenant_item = TenantMRR.from_dict(by_tenant_item_data)

                by_tenant.append(by_tenant_item)

        currency_mrr = cls(
            total_mrr=total_mrr,
            currency=currency,
            converted_mrr=converted_mrr,
            rate=rate,
            fx_error=fx_error,
            by_tenant=by_tenant,
        )

        currency_mrr.additional_properties = d
        return currency_mrr

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
