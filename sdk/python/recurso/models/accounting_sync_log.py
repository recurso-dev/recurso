from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AccountingSyncLog")


@_attrs_define
class AccountingSyncLog:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        connection_id (UUID | Unset):
        entity_type (str | Unset):
        entity_id (UUID | Unset):
        external_id (str | Unset): ID assigned by the accounting provider.
        action (str | Unset):
        status (str | Unset):
        error_message (str | Unset):
        synced_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    connection_id: UUID | Unset = UNSET
    entity_type: str | Unset = UNSET
    entity_id: UUID | Unset = UNSET
    external_id: str | Unset = UNSET
    action: str | Unset = UNSET
    status: str | Unset = UNSET
    error_message: str | Unset = UNSET
    synced_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        connection_id: str | Unset = UNSET
        if not isinstance(self.connection_id, Unset):
            connection_id = str(self.connection_id)

        entity_type = self.entity_type

        entity_id: str | Unset = UNSET
        if not isinstance(self.entity_id, Unset):
            entity_id = str(self.entity_id)

        external_id = self.external_id

        action = self.action

        status = self.status

        error_message = self.error_message

        synced_at: str | Unset = UNSET
        if not isinstance(self.synced_at, Unset):
            synced_at = self.synced_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if connection_id is not UNSET:
            field_dict["connection_id"] = connection_id
        if entity_type is not UNSET:
            field_dict["entity_type"] = entity_type
        if entity_id is not UNSET:
            field_dict["entity_id"] = entity_id
        if external_id is not UNSET:
            field_dict["external_id"] = external_id
        if action is not UNSET:
            field_dict["action"] = action
        if status is not UNSET:
            field_dict["status"] = status
        if error_message is not UNSET:
            field_dict["error_message"] = error_message
        if synced_at is not UNSET:
            field_dict["synced_at"] = synced_at

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

        _connection_id = d.pop("connection_id", UNSET)
        connection_id: UUID | Unset
        if isinstance(_connection_id, Unset):
            connection_id = UNSET
        else:
            connection_id = UUID(_connection_id)

        entity_type = d.pop("entity_type", UNSET)

        _entity_id = d.pop("entity_id", UNSET)
        entity_id: UUID | Unset
        if isinstance(_entity_id, Unset):
            entity_id = UNSET
        else:
            entity_id = UUID(_entity_id)

        external_id = d.pop("external_id", UNSET)

        action = d.pop("action", UNSET)

        status = d.pop("status", UNSET)

        error_message = d.pop("error_message", UNSET)

        _synced_at = d.pop("synced_at", UNSET)
        synced_at: datetime.datetime | Unset
        if isinstance(_synced_at, Unset):
            synced_at = UNSET
        else:
            synced_at = datetime.datetime.fromisoformat(_synced_at)

        accounting_sync_log = cls(
            id=id,
            tenant_id=tenant_id,
            connection_id=connection_id,
            entity_type=entity_type,
            entity_id=entity_id,
            external_id=external_id,
            action=action,
            status=status,
            error_message=error_message,
            synced_at=synced_at,
        )

        accounting_sync_log.additional_properties = d
        return accounting_sync_log

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
