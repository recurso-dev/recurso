from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="APIKey")


@_attrs_define
class APIKey:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        key_value (str | Unset): Raw API key. Populated only in the creation response — never on list.
        key_prefix (str | Unset): First characters of the key, for display and lookup.
        type_ (str | Unset):
        is_active (bool | Unset):
        livemode (bool | Unset): true for a live key (rsk_live_), false for a test key (rsk_test_).
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    key_value: str | Unset = UNSET
    key_prefix: str | Unset = UNSET
    type_: str | Unset = UNSET
    is_active: bool | Unset = UNSET
    livemode: bool | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        key_value = self.key_value

        key_prefix = self.key_prefix

        type_ = self.type_

        is_active = self.is_active

        livemode = self.livemode

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
        if key_value is not UNSET:
            field_dict["key_value"] = key_value
        if key_prefix is not UNSET:
            field_dict["key_prefix"] = key_prefix
        if type_ is not UNSET:
            field_dict["type"] = type_
        if is_active is not UNSET:
            field_dict["is_active"] = is_active
        if livemode is not UNSET:
            field_dict["livemode"] = livemode
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

        key_value = d.pop("key_value", UNSET)

        key_prefix = d.pop("key_prefix", UNSET)

        type_ = d.pop("type", UNSET)

        is_active = d.pop("is_active", UNSET)

        livemode = d.pop("livemode", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        api_key = cls(
            id=id,
            tenant_id=tenant_id,
            key_value=key_value,
            key_prefix=key_prefix,
            type_=type_,
            is_active=is_active,
            livemode=livemode,
            created_at=created_at,
        )

        api_key.additional_properties = d
        return api_key

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
