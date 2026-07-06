from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.unbilled_charge_status import UnbilledChargeStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="UnbilledCharge")


@_attrs_define
class UnbilledCharge:
    """
    Attributes:
        id (UUID | Unset):
        subscription_id (UUID | Unset):
        amount (int | Unset): Amount in the lowest currency unit.
        currency (str | Unset):
        description (str | Unset):
        status (UnbilledChargeStatus | Unset):
        period_start (datetime.datetime | Unset):
        period_end (datetime.datetime | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    subscription_id: UUID | Unset = UNSET
    amount: int | Unset = UNSET
    currency: str | Unset = UNSET
    description: str | Unset = UNSET
    status: UnbilledChargeStatus | Unset = UNSET
    period_start: datetime.datetime | Unset = UNSET
    period_end: datetime.datetime | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        subscription_id: str | Unset = UNSET
        if not isinstance(self.subscription_id, Unset):
            subscription_id = str(self.subscription_id)

        amount = self.amount

        currency = self.currency

        description = self.description

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        period_start: str | Unset = UNSET
        if not isinstance(self.period_start, Unset):
            period_start = self.period_start.isoformat()

        period_end: str | Unset = UNSET
        if not isinstance(self.period_end, Unset):
            period_end = self.period_end.isoformat()

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if subscription_id is not UNSET:
            field_dict["subscription_id"] = subscription_id
        if amount is not UNSET:
            field_dict["amount"] = amount
        if currency is not UNSET:
            field_dict["currency"] = currency
        if description is not UNSET:
            field_dict["description"] = description
        if status is not UNSET:
            field_dict["status"] = status
        if period_start is not UNSET:
            field_dict["period_start"] = period_start
        if period_end is not UNSET:
            field_dict["period_end"] = period_end
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

        _subscription_id = d.pop("subscription_id", UNSET)
        subscription_id: UUID | Unset
        if isinstance(_subscription_id, Unset):
            subscription_id = UNSET
        else:
            subscription_id = UUID(_subscription_id)

        amount = d.pop("amount", UNSET)

        currency = d.pop("currency", UNSET)

        description = d.pop("description", UNSET)

        _status = d.pop("status", UNSET)
        status: UnbilledChargeStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = UnbilledChargeStatus(_status)

        _period_start = d.pop("period_start", UNSET)
        period_start: datetime.datetime | Unset
        if isinstance(_period_start, Unset):
            period_start = UNSET
        else:
            period_start = datetime.datetime.fromisoformat(_period_start)

        _period_end = d.pop("period_end", UNSET)
        period_end: datetime.datetime | Unset
        if isinstance(_period_end, Unset):
            period_end = UNSET
        else:
            period_end = datetime.datetime.fromisoformat(_period_end)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        unbilled_charge = cls(
            id=id,
            subscription_id=subscription_id,
            amount=amount,
            currency=currency,
            description=description,
            status=status,
            period_start=period_start,
            period_end=period_end,
            created_at=created_at,
        )

        unbilled_charge.additional_properties = d
        return unbilled_charge

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
