from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.accounting_connection_provider import AccountingConnectionProvider
from ..types import UNSET, Unset

T = TypeVar("T", bound="AccountingConnection")


@_attrs_define
class AccountingConnection:
    """OAuth access/refresh tokens are stored server-side and never serialized.

    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        provider (AccountingConnectionProvider | Unset):
        token_expires_at (datetime.datetime | None | Unset):
        realm_id (str | Unset): QuickBooks company ID / Xero organisation ID.
        last_sync_at (datetime.datetime | None | Unset):
        sync_status (str | Unset):
        last_error (str | Unset):
        is_active (bool | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    provider: AccountingConnectionProvider | Unset = UNSET
    token_expires_at: datetime.datetime | None | Unset = UNSET
    realm_id: str | Unset = UNSET
    last_sync_at: datetime.datetime | None | Unset = UNSET
    sync_status: str | Unset = UNSET
    last_error: str | Unset = UNSET
    is_active: bool | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        provider: str | Unset = UNSET
        if not isinstance(self.provider, Unset):
            provider = self.provider.value

        token_expires_at: None | str | Unset
        if isinstance(self.token_expires_at, Unset):
            token_expires_at = UNSET
        elif isinstance(self.token_expires_at, datetime.datetime):
            token_expires_at = self.token_expires_at.isoformat()
        else:
            token_expires_at = self.token_expires_at

        realm_id = self.realm_id

        last_sync_at: None | str | Unset
        if isinstance(self.last_sync_at, Unset):
            last_sync_at = UNSET
        elif isinstance(self.last_sync_at, datetime.datetime):
            last_sync_at = self.last_sync_at.isoformat()
        else:
            last_sync_at = self.last_sync_at

        sync_status = self.sync_status

        last_error = self.last_error

        is_active = self.is_active

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if provider is not UNSET:
            field_dict["provider"] = provider
        if token_expires_at is not UNSET:
            field_dict["token_expires_at"] = token_expires_at
        if realm_id is not UNSET:
            field_dict["realm_id"] = realm_id
        if last_sync_at is not UNSET:
            field_dict["last_sync_at"] = last_sync_at
        if sync_status is not UNSET:
            field_dict["sync_status"] = sync_status
        if last_error is not UNSET:
            field_dict["last_error"] = last_error
        if is_active is not UNSET:
            field_dict["is_active"] = is_active
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _id = d.pop("id", UNSET)
        id: UUID | Unset
        if isinstance(_id, Unset):
            id = UNSET
        else:
            id = UUID(_id)

        _tenant_id = d.pop("tenant_id", UNSET)
        tenant_id: UUID | Unset
        if isinstance(_tenant_id, Unset):
            tenant_id = UNSET
        else:
            tenant_id = UUID(_tenant_id)

        _provider = d.pop("provider", UNSET)
        provider: AccountingConnectionProvider | Unset
        if isinstance(_provider, Unset):
            provider = UNSET
        else:
            provider = AccountingConnectionProvider(_provider)

        def _parse_token_expires_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                token_expires_at_type_0 = datetime.datetime.fromisoformat(data)

                return token_expires_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        token_expires_at = _parse_token_expires_at(d.pop("token_expires_at", UNSET))

        realm_id = d.pop("realm_id", UNSET)

        def _parse_last_sync_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_sync_at_type_0 = datetime.datetime.fromisoformat(data)

                return last_sync_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        last_sync_at = _parse_last_sync_at(d.pop("last_sync_at", UNSET))

        sync_status = d.pop("sync_status", UNSET)

        last_error = d.pop("last_error", UNSET)

        is_active = d.pop("is_active", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        accounting_connection = cls(
            id=id,
            tenant_id=tenant_id,
            provider=provider,
            token_expires_at=token_expires_at,
            realm_id=realm_id,
            last_sync_at=last_sync_at,
            sync_status=sync_status,
            last_error=last_error,
            is_active=is_active,
            created_at=created_at,
        )

        accounting_connection.additional_properties = d
        return accounting_connection

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
