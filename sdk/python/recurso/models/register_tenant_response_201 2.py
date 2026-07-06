from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.tenant import Tenant


T = TypeVar("T", bound="RegisterTenantResponse201")


@_attrs_define
class RegisterTenantResponse201:
    """
    Attributes:
        tenant (Tenant | Unset):
        api_key (str | Unset): The raw API key. Store it securely — it is not retrievable later.
    """

    tenant: Tenant | Unset = UNSET
    api_key: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        tenant: dict[str, Any] | Unset = UNSET
        if not isinstance(self.tenant, Unset):
            tenant = self.tenant.to_dict()

        api_key = self.api_key

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if tenant is not UNSET:
            field_dict["tenant"] = tenant
        if api_key is not UNSET:
            field_dict["api_key"] = api_key

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tenant import Tenant

        d = dict(src_dict)
        _tenant = d.pop("tenant", UNSET)
        tenant: Tenant | Unset
        if isinstance(_tenant, Unset):
            tenant = UNSET
        else:
            tenant = Tenant.from_dict(_tenant)

        api_key = d.pop("api_key", UNSET)

        register_tenant_response_201 = cls(
            tenant=tenant,
            api_key=api_key,
        )

        register_tenant_response_201.additional_properties = d
        return register_tenant_response_201

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
