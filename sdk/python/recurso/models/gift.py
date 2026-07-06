from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.gift_status import GiftStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="Gift")


@_attrs_define
class Gift:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        code (str | Unset): Redemption code.
        plan_id (UUID | Unset):
        buyer_customer_id (UUID | Unset):
        recipient_email (str | Unset):
        status (GiftStatus | Unset):
        redeemed_by_customer_id (None | Unset | UUID):
        redeemed_at (datetime.datetime | None | Unset):
        duration_months (int | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    code: str | Unset = UNSET
    plan_id: UUID | Unset = UNSET
    buyer_customer_id: UUID | Unset = UNSET
    recipient_email: str | Unset = UNSET
    status: GiftStatus | Unset = UNSET
    redeemed_by_customer_id: None | Unset | UUID = UNSET
    redeemed_at: datetime.datetime | None | Unset = UNSET
    duration_months: int | Unset = UNSET
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

        code = self.code

        plan_id: str | Unset = UNSET
        if not isinstance(self.plan_id, Unset):
            plan_id = str(self.plan_id)

        buyer_customer_id: str | Unset = UNSET
        if not isinstance(self.buyer_customer_id, Unset):
            buyer_customer_id = str(self.buyer_customer_id)

        recipient_email = self.recipient_email

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        redeemed_by_customer_id: None | str | Unset
        if isinstance(self.redeemed_by_customer_id, Unset):
            redeemed_by_customer_id = UNSET
        elif isinstance(self.redeemed_by_customer_id, UUID):
            redeemed_by_customer_id = str(self.redeemed_by_customer_id)
        else:
            redeemed_by_customer_id = self.redeemed_by_customer_id

        redeemed_at: None | str | Unset
        if isinstance(self.redeemed_at, Unset):
            redeemed_at = UNSET
        elif isinstance(self.redeemed_at, datetime.datetime):
            redeemed_at = self.redeemed_at.isoformat()
        else:
            redeemed_at = self.redeemed_at

        duration_months = self.duration_months

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
        if code is not UNSET:
            field_dict["code"] = code
        if plan_id is not UNSET:
            field_dict["plan_id"] = plan_id
        if buyer_customer_id is not UNSET:
            field_dict["buyer_customer_id"] = buyer_customer_id
        if recipient_email is not UNSET:
            field_dict["recipient_email"] = recipient_email
        if status is not UNSET:
            field_dict["status"] = status
        if redeemed_by_customer_id is not UNSET:
            field_dict["redeemed_by_customer_id"] = redeemed_by_customer_id
        if redeemed_at is not UNSET:
            field_dict["redeemed_at"] = redeemed_at
        if duration_months is not UNSET:
            field_dict["duration_months"] = duration_months
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

        code = d.pop("code", UNSET)

        _plan_id = d.pop("plan_id", UNSET)
        plan_id: UUID | Unset
        if isinstance(_plan_id, Unset):
            plan_id = UNSET
        else:
            plan_id = UUID(_plan_id)

        _buyer_customer_id = d.pop("buyer_customer_id", UNSET)
        buyer_customer_id: UUID | Unset
        if isinstance(_buyer_customer_id, Unset):
            buyer_customer_id = UNSET
        else:
            buyer_customer_id = UUID(_buyer_customer_id)

        recipient_email = d.pop("recipient_email", UNSET)

        _status = d.pop("status", UNSET)
        status: GiftStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = GiftStatus(_status)

        def _parse_redeemed_by_customer_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                redeemed_by_customer_id_type_0 = UUID(data)

                return redeemed_by_customer_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        redeemed_by_customer_id = _parse_redeemed_by_customer_id(d.pop("redeemed_by_customer_id", UNSET))

        def _parse_redeemed_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                redeemed_at_type_0 = datetime.datetime.fromisoformat(data)

                return redeemed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        redeemed_at = _parse_redeemed_at(d.pop("redeemed_at", UNSET))

        duration_months = d.pop("duration_months", UNSET)

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

        gift = cls(
            id=id,
            tenant_id=tenant_id,
            code=code,
            plan_id=plan_id,
            buyer_customer_id=buyer_customer_id,
            recipient_email=recipient_email,
            status=status,
            redeemed_by_customer_id=redeemed_by_customer_id,
            redeemed_at=redeemed_at,
            duration_months=duration_months,
            created_at=created_at,
            updated_at=updated_at,
        )

        gift.additional_properties = d
        return gift

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
