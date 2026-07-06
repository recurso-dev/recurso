from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.referral_status import ReferralStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="Referral")


@_attrs_define
class Referral:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        referrer_id (UUID | Unset):
        referred_id (UUID | Unset):
        code (str | Unset):
        status (ReferralStatus | Unset):
        reward_amount (int | Unset):
        currency (str | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
        qualified_at (datetime.datetime | None | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    referrer_id: UUID | Unset = UNSET
    referred_id: UUID | Unset = UNSET
    code: str | Unset = UNSET
    status: ReferralStatus | Unset = UNSET
    reward_amount: int | Unset = UNSET
    currency: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    qualified_at: datetime.datetime | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        referrer_id: str | Unset = UNSET
        if not isinstance(self.referrer_id, Unset):
            referrer_id = str(self.referrer_id)

        referred_id: str | Unset = UNSET
        if not isinstance(self.referred_id, Unset):
            referred_id = str(self.referred_id)

        code = self.code

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        reward_amount = self.reward_amount

        currency = self.currency

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        qualified_at: None | str | Unset
        if isinstance(self.qualified_at, Unset):
            qualified_at = UNSET
        elif isinstance(self.qualified_at, datetime.datetime):
            qualified_at = self.qualified_at.isoformat()
        else:
            qualified_at = self.qualified_at

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if referrer_id is not UNSET:
            field_dict["referrer_id"] = referrer_id
        if referred_id is not UNSET:
            field_dict["referred_id"] = referred_id
        if code is not UNSET:
            field_dict["code"] = code
        if status is not UNSET:
            field_dict["status"] = status
        if reward_amount is not UNSET:
            field_dict["reward_amount"] = reward_amount
        if currency is not UNSET:
            field_dict["currency"] = currency
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at
        if qualified_at is not UNSET:
            field_dict["qualified_at"] = qualified_at

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

        _referrer_id = d.pop("referrer_id", UNSET)
        referrer_id: UUID | Unset
        if isinstance(_referrer_id, Unset):
            referrer_id = UNSET
        else:
            referrer_id = UUID(_referrer_id)

        _referred_id = d.pop("referred_id", UNSET)
        referred_id: UUID | Unset
        if isinstance(_referred_id, Unset):
            referred_id = UNSET
        else:
            referred_id = UUID(_referred_id)

        code = d.pop("code", UNSET)

        _status = d.pop("status", UNSET)
        status: ReferralStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = ReferralStatus(_status)

        reward_amount = d.pop("reward_amount", UNSET)

        currency = d.pop("currency", UNSET)

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

        def _parse_qualified_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                qualified_at_type_0 = datetime.datetime.fromisoformat(data)

                return qualified_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        qualified_at = _parse_qualified_at(d.pop("qualified_at", UNSET))

        referral = cls(
            id=id,
            tenant_id=tenant_id,
            referrer_id=referrer_id,
            referred_id=referred_id,
            code=code,
            status=status,
            reward_amount=reward_amount,
            currency=currency,
            created_at=created_at,
            updated_at=updated_at,
            qualified_at=qualified_at,
        )

        referral.additional_properties = d
        return referral

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
