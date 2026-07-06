from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.ledger_account_type import LedgerAccountType
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ledger_account_user_data_128 import LedgerAccountUserData128


T = TypeVar("T", bound="LedgerAccount")


@_attrs_define
class LedgerAccount:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        name (str | Unset):
        type_ (LedgerAccountType | Unset):
        code (int | Unset): Chart-of-accounts code (e.g. 1000 for Cash).
        ledger_id (int | Unset):
        user_data_128 (LedgerAccountUserData128 | Unset): Internal TigerBeetle user data (opaque).
        credits_posted (int | Unset):
        debits_posted (int | Unset):
        currency (str | Unset):
        balance (int | Unset): Cached balance snapshot in the lowest currency unit.
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    name: str | Unset = UNSET
    type_: LedgerAccountType | Unset = UNSET
    code: int | Unset = UNSET
    ledger_id: int | Unset = UNSET
    user_data_128: LedgerAccountUserData128 | Unset = UNSET
    credits_posted: int | Unset = UNSET
    debits_posted: int | Unset = UNSET
    currency: str | Unset = UNSET
    balance: int | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        name = self.name

        type_: str | Unset = UNSET
        if not isinstance(self.type_, Unset):
            type_ = self.type_.value

        code = self.code

        ledger_id = self.ledger_id

        user_data_128: dict[str, Any] | Unset = UNSET
        if not isinstance(self.user_data_128, Unset):
            user_data_128 = self.user_data_128.to_dict()

        credits_posted = self.credits_posted

        debits_posted = self.debits_posted

        currency = self.currency

        balance = self.balance

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
        if name is not UNSET:
            field_dict["name"] = name
        if type_ is not UNSET:
            field_dict["type"] = type_
        if code is not UNSET:
            field_dict["code"] = code
        if ledger_id is not UNSET:
            field_dict["ledger_id"] = ledger_id
        if user_data_128 is not UNSET:
            field_dict["user_data_128"] = user_data_128
        if credits_posted is not UNSET:
            field_dict["credits_posted"] = credits_posted
        if debits_posted is not UNSET:
            field_dict["debits_posted"] = debits_posted
        if currency is not UNSET:
            field_dict["currency"] = currency
        if balance is not UNSET:
            field_dict["balance"] = balance
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ledger_account_user_data_128 import LedgerAccountUserData128

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

        name = d.pop("name", UNSET)

        _type_ = d.pop("type", UNSET)
        type_: LedgerAccountType | Unset
        if isinstance(_type_, Unset):
            type_ = UNSET
        else:
            type_ = LedgerAccountType(_type_)

        code = d.pop("code", UNSET)

        ledger_id = d.pop("ledger_id", UNSET)

        _user_data_128 = d.pop("user_data_128", UNSET)
        user_data_128: LedgerAccountUserData128 | Unset
        if isinstance(_user_data_128, Unset):
            user_data_128 = UNSET
        else:
            user_data_128 = LedgerAccountUserData128.from_dict(_user_data_128)

        credits_posted = d.pop("credits_posted", UNSET)

        debits_posted = d.pop("debits_posted", UNSET)

        currency = d.pop("currency", UNSET)

        balance = d.pop("balance", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        ledger_account = cls(
            id=id,
            tenant_id=tenant_id,
            name=name,
            type_=type_,
            code=code,
            ledger_id=ledger_id,
            user_data_128=user_data_128,
            credits_posted=credits_posted,
            debits_posted=debits_posted,
            currency=currency,
            balance=balance,
            created_at=created_at,
        )

        ledger_account.additional_properties = d
        return ledger_account

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
