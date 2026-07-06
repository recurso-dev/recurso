from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.price_type import PriceType
from ..types import UNSET, Unset

T = TypeVar("T", bound="Price")


@_attrs_define
class Price:
    """
    Attributes:
        id (UUID | Unset):
        plan_id (UUID | Unset):
        currency (str | Unset): ISO 4217 code.
        amount (int | Unset): Price in the lowest currency unit.
        type_ (PriceType | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    plan_id: UUID | Unset = UNSET
    currency: str | Unset = UNSET
    amount: int | Unset = UNSET
    type_: PriceType | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        plan_id: str | Unset = UNSET
        if not isinstance(self.plan_id, Unset):
            plan_id = str(self.plan_id)

        currency = self.currency

        amount = self.amount

        type_: str | Unset = UNSET
        if not isinstance(self.type_, Unset):
            type_ = self.type_.value

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if plan_id is not UNSET:
            field_dict["plan_id"] = plan_id
        if currency is not UNSET:
            field_dict["currency"] = currency
        if amount is not UNSET:
            field_dict["amount"] = amount
        if type_ is not UNSET:
            field_dict["type"] = type_
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

        _plan_id = d.pop("plan_id", UNSET)
        plan_id: UUID | Unset
        if isinstance(_plan_id, Unset):
            plan_id = UNSET
        else:
            plan_id = UUID(_plan_id)

        currency = d.pop("currency", UNSET)

        amount = d.pop("amount", UNSET)

        _type_ = d.pop("type", UNSET)
        type_: PriceType | Unset
        if isinstance(_type_, Unset):
            type_ = UNSET
        else:
            type_ = PriceType(_type_)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        price = cls(
            id=id,
            plan_id=plan_id,
            currency=currency,
            amount=amount,
            type_=type_,
            created_at=created_at,
        )

        price.additional_properties = d
        return price

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
