from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateReferralBody")


@_attrs_define
class CreateReferralBody:
    """
    Attributes:
        referrer_id (UUID):
        referred_id (UUID):
        reward_amount (int | Unset): Reward in the lowest currency unit (defaults to 500 = $5.00). Default: 500.
        currency (str | Unset):  Default: 'USD'.
    """

    referrer_id: UUID
    referred_id: UUID
    reward_amount: int | Unset = 500
    currency: str | Unset = "USD"
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        referrer_id = str(self.referrer_id)

        referred_id = str(self.referred_id)

        reward_amount = self.reward_amount

        currency = self.currency

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "referrer_id": referrer_id,
                "referred_id": referred_id,
            }
        )
        if reward_amount is not UNSET:
            field_dict["reward_amount"] = reward_amount
        if currency is not UNSET:
            field_dict["currency"] = currency

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        referrer_id = UUID(d.pop("referrer_id"))

        referred_id = UUID(d.pop("referred_id"))

        reward_amount = d.pop("reward_amount", UNSET)

        currency = d.pop("currency", UNSET)

        create_referral_body = cls(
            referrer_id=referrer_id,
            referred_id=referred_id,
            reward_amount=reward_amount,
            currency=currency,
        )

        create_referral_body.additional_properties = d
        return create_referral_body

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
