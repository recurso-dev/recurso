from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.consent_consent_type import ConsentConsentType
from ..types import UNSET, Unset

T = TypeVar("T", bound="Consent")


@_attrs_define
class Consent:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        subscription_id (None | Unset | UUID):
        consent_type (ConsentConsentType | Unset):
        granted (bool | Unset):
        granted_at (datetime.datetime | Unset):
        revoked_at (datetime.datetime | None | Unset):
        ip_address (str | Unset):
        user_agent (str | Unset):
        consent_text (str | Unset): Exact text shown to the customer.
        version (str | Unset): Version of the consent text.
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    subscription_id: None | Unset | UUID = UNSET
    consent_type: ConsentConsentType | Unset = UNSET
    granted: bool | Unset = UNSET
    granted_at: datetime.datetime | Unset = UNSET
    revoked_at: datetime.datetime | None | Unset = UNSET
    ip_address: str | Unset = UNSET
    user_agent: str | Unset = UNSET
    consent_text: str | Unset = UNSET
    version: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        subscription_id: None | str | Unset
        if isinstance(self.subscription_id, Unset):
            subscription_id = UNSET
        elif isinstance(self.subscription_id, UUID):
            subscription_id = str(self.subscription_id)
        else:
            subscription_id = self.subscription_id

        consent_type: str | Unset = UNSET
        if not isinstance(self.consent_type, Unset):
            consent_type = self.consent_type.value

        granted = self.granted

        granted_at: str | Unset = UNSET
        if not isinstance(self.granted_at, Unset):
            granted_at = self.granted_at.isoformat()

        revoked_at: None | str | Unset
        if isinstance(self.revoked_at, Unset):
            revoked_at = UNSET
        elif isinstance(self.revoked_at, datetime.datetime):
            revoked_at = self.revoked_at.isoformat()
        else:
            revoked_at = self.revoked_at

        ip_address = self.ip_address

        user_agent = self.user_agent

        consent_text = self.consent_text

        version = self.version

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if subscription_id is not UNSET:
            field_dict["subscription_id"] = subscription_id
        if consent_type is not UNSET:
            field_dict["consent_type"] = consent_type
        if granted is not UNSET:
            field_dict["granted"] = granted
        if granted_at is not UNSET:
            field_dict["granted_at"] = granted_at
        if revoked_at is not UNSET:
            field_dict["revoked_at"] = revoked_at
        if ip_address is not UNSET:
            field_dict["ip_address"] = ip_address
        if user_agent is not UNSET:
            field_dict["user_agent"] = user_agent
        if consent_text is not UNSET:
            field_dict["consent_text"] = consent_text
        if version is not UNSET:
            field_dict["version"] = version
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at

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

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        def _parse_subscription_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                subscription_id_type_0 = UUID(data)

                return subscription_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        subscription_id = _parse_subscription_id(d.pop("subscription_id", UNSET))

        _consent_type = d.pop("consent_type", UNSET)
        consent_type: ConsentConsentType | Unset
        if isinstance(_consent_type, Unset):
            consent_type = UNSET
        else:
            consent_type = ConsentConsentType(_consent_type)

        granted = d.pop("granted", UNSET)

        _granted_at = d.pop("granted_at", UNSET)
        granted_at: datetime.datetime | Unset
        if isinstance(_granted_at, Unset):
            granted_at = UNSET
        else:
            granted_at = datetime.datetime.fromisoformat(_granted_at)

        def _parse_revoked_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                revoked_at_type_0 = datetime.datetime.fromisoformat(data)

                return revoked_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        revoked_at = _parse_revoked_at(d.pop("revoked_at", UNSET))

        ip_address = d.pop("ip_address", UNSET)

        user_agent = d.pop("user_agent", UNSET)

        consent_text = d.pop("consent_text", UNSET)

        version = d.pop("version", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        _updated_at = d.pop("updated_at", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at, Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)

        consent = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            subscription_id=subscription_id,
            consent_type=consent_type,
            granted=granted,
            granted_at=granted_at,
            revoked_at=revoked_at,
            ip_address=ip_address,
            user_agent=user_agent,
            consent_text=consent_text,
            version=version,
            created_at=created_at,
            updated_at=updated_at,
        )

        consent.additional_properties = d
        return consent

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
