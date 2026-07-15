from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.get_current_user_response_200_tenant import GetCurrentUserResponse200Tenant
    from ..models.user import User


T = TypeVar("T", bound="GetCurrentUserResponse200")


@_attrs_define
class GetCurrentUserResponse200:
    """
    Attributes:
        user (User | Unset): A dashboard user account within a tenant.
        tenant (GetCurrentUserResponse200Tenant | Unset):
    """

    user: User | Unset = UNSET
    tenant: GetCurrentUserResponse200Tenant | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        user: dict[str, Any] | Unset = UNSET
        if not isinstance(self.user, Unset):
            user = self.user.to_dict()

        tenant: dict[str, Any] | Unset = UNSET
        if not isinstance(self.tenant, Unset):
            tenant = self.tenant.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if user is not UNSET:
            field_dict["user"] = user
        if tenant is not UNSET:
            field_dict["tenant"] = tenant

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.get_current_user_response_200_tenant import GetCurrentUserResponse200Tenant
        from ..models.user import User

        d = dict(src_dict)
        _user = d.pop("user", UNSET)
        user: User | Unset
        if isinstance(_user, Unset):
            user = UNSET
        else:
            user = User.from_dict(_user)

        _tenant = d.pop("tenant", UNSET)
        tenant: GetCurrentUserResponse200Tenant | Unset
        if isinstance(_tenant, Unset):
            tenant = UNSET
        else:
            tenant = GetCurrentUserResponse200Tenant.from_dict(_tenant)

        get_current_user_response_200 = cls(
            user=user,
            tenant=tenant,
        )

        get_current_user_response_200.additional_properties = d
        return get_current_user_response_200

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
