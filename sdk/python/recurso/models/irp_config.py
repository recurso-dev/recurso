from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.irp_config_environment import IRPConfigEnvironment
from ..types import UNSET, Unset

T = TypeVar("T", bound="IRPConfig")


@_attrs_define
class IRPConfig:
    """
    Attributes:
        id (str | Unset):
        tenant_id (str | Unset):
        environment (IRPConfigEnvironment | Unset):
        client_id (str | Unset):
        client_secret (str | Unset): Masked (first 4 characters + ****) in GET responses.
        username (str | Unset):
        password (str | Unset): Masked (****) in GET responses.
        gstin (str | Unset):
        is_enabled (bool | Unset):
    """

    id: str | Unset = UNSET
    tenant_id: str | Unset = UNSET
    environment: IRPConfigEnvironment | Unset = UNSET
    client_id: str | Unset = UNSET
    client_secret: str | Unset = UNSET
    username: str | Unset = UNSET
    password: str | Unset = UNSET
    gstin: str | Unset = UNSET
    is_enabled: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        tenant_id = self.tenant_id

        environment: str | Unset = UNSET
        if not isinstance(self.environment, Unset):
            environment = self.environment.value

        client_id = self.client_id

        client_secret = self.client_secret

        username = self.username

        password = self.password

        gstin = self.gstin

        is_enabled = self.is_enabled

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if environment is not UNSET:
            field_dict["environment"] = environment
        if client_id is not UNSET:
            field_dict["client_id"] = client_id
        if client_secret is not UNSET:
            field_dict["client_secret"] = client_secret
        if username is not UNSET:
            field_dict["username"] = username
        if password is not UNSET:
            field_dict["password"] = password
        if gstin is not UNSET:
            field_dict["gstin"] = gstin
        if is_enabled is not UNSET:
            field_dict["is_enabled"] = is_enabled

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id", UNSET)

        tenant_id = d.pop("tenant_id", UNSET)

        _environment = d.pop("environment", UNSET)
        environment: IRPConfigEnvironment | Unset
        if isinstance(_environment, Unset):
            environment = UNSET
        else:
            environment = IRPConfigEnvironment(_environment)

        client_id = d.pop("client_id", UNSET)

        client_secret = d.pop("client_secret", UNSET)

        username = d.pop("username", UNSET)

        password = d.pop("password", UNSET)

        gstin = d.pop("gstin", UNSET)

        is_enabled = d.pop("is_enabled", UNSET)

        irp_config = cls(
            id=id,
            tenant_id=tenant_id,
            environment=environment,
            client_id=client_id,
            client_secret=client_secret,
            username=username,
            password=password,
            gstin=gstin,
            is_enabled=is_enabled,
        )

        irp_config.additional_properties = d
        return irp_config

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
