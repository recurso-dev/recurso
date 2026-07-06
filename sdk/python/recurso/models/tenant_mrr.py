from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="TenantMRR")


@_attrs_define
class TenantMRR:
    """
    Attributes:
        tenant_id (UUID | Unset):
        tenant_name (str | Unset):
        mrr (int | Unset):
        currency (str | Unset):
    """

    tenant_id: UUID | Unset = UNSET
    tenant_name: str | Unset = UNSET
    mrr: int | Unset = UNSET
    currency: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        tenant_name = self.tenant_name

        mrr = self.mrr

        currency = self.currency

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if tenant_name is not UNSET:
            field_dict["tenant_name"] = tenant_name
        if mrr is not UNSET:
            field_dict["mrr"] = mrr
        if currency is not UNSET:
            field_dict["currency"] = currency

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _tenant_id = d.pop("tenant_id", UNSET)
        tenant_id: UUID | Unset
        if isinstance(_tenant_id, Unset):
            tenant_id = UNSET
        else:
            tenant_id = UUID(_tenant_id)

        tenant_name = d.pop("tenant_name", UNSET)

        mrr = d.pop("mrr", UNSET)

        currency = d.pop("currency", UNSET)

        tenant_mrr = cls(
            tenant_id=tenant_id,
            tenant_name=tenant_name,
            mrr=mrr,
            currency=currency,
        )

        tenant_mrr.additional_properties = d
        return tenant_mrr

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
