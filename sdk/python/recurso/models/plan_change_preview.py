from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PlanChangePreview")


@_attrs_define
class PlanChangePreview:
    """Read-only proration breakdown for a subscription plan change. Monetary fields are in the currency's smallest unit
    (e.g. paise/cents).

        Attributes:
            subscription_id (UUID | Unset):
            current_plan_id (UUID | Unset):
            new_plan_id (UUID | Unset):
            currency (str | Unset):
            credit_amount (int | Unset): Credit for unused time on the current plan.
            charge_amount (int | Unset): Prorated charge for the remaining period on the new plan.
            net_amount (int | Unset): charge_amount - credit_amount, before tax.
            tax_amount (int | Unset): Tax on a positive net (0 for credits).
            total_amount (int | Unset): net_amount + tax_amount; the immediate proration invoice total.
            effective_date (datetime.datetime | Unset):
            next_invoice_amount (int | Unset): Full new-plan charge incl. tax at the next renewal.
            is_upgrade (bool | Unset):
    """

    subscription_id: UUID | Unset = UNSET
    current_plan_id: UUID | Unset = UNSET
    new_plan_id: UUID | Unset = UNSET
    currency: str | Unset = UNSET
    credit_amount: int | Unset = UNSET
    charge_amount: int | Unset = UNSET
    net_amount: int | Unset = UNSET
    tax_amount: int | Unset = UNSET
    total_amount: int | Unset = UNSET
    effective_date: datetime.datetime | Unset = UNSET
    next_invoice_amount: int | Unset = UNSET
    is_upgrade: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        subscription_id: str | Unset = UNSET
        if not isinstance(self.subscription_id, Unset):
            subscription_id = str(self.subscription_id)

        current_plan_id: str | Unset = UNSET
        if not isinstance(self.current_plan_id, Unset):
            current_plan_id = str(self.current_plan_id)

        new_plan_id: str | Unset = UNSET
        if not isinstance(self.new_plan_id, Unset):
            new_plan_id = str(self.new_plan_id)

        currency = self.currency

        credit_amount = self.credit_amount

        charge_amount = self.charge_amount

        net_amount = self.net_amount

        tax_amount = self.tax_amount

        total_amount = self.total_amount

        effective_date: str | Unset = UNSET
        if not isinstance(self.effective_date, Unset):
            effective_date = self.effective_date.isoformat()

        next_invoice_amount = self.next_invoice_amount

        is_upgrade = self.is_upgrade

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if subscription_id is not UNSET:
            field_dict["subscription_id"] = subscription_id
        if current_plan_id is not UNSET:
            field_dict["current_plan_id"] = current_plan_id
        if new_plan_id is not UNSET:
            field_dict["new_plan_id"] = new_plan_id
        if currency is not UNSET:
            field_dict["currency"] = currency
        if credit_amount is not UNSET:
            field_dict["credit_amount"] = credit_amount
        if charge_amount is not UNSET:
            field_dict["charge_amount"] = charge_amount
        if net_amount is not UNSET:
            field_dict["net_amount"] = net_amount
        if tax_amount is not UNSET:
            field_dict["tax_amount"] = tax_amount
        if total_amount is not UNSET:
            field_dict["total_amount"] = total_amount
        if effective_date is not UNSET:
            field_dict["effective_date"] = effective_date
        if next_invoice_amount is not UNSET:
            field_dict["next_invoice_amount"] = next_invoice_amount
        if is_upgrade is not UNSET:
            field_dict["is_upgrade"] = is_upgrade

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _subscription_id = d.pop("subscription_id", UNSET)
        subscription_id: UUID | Unset
        if isinstance(_subscription_id, Unset):
            subscription_id = UNSET
        else:
            subscription_id = UUID(_subscription_id)

        _current_plan_id = d.pop("current_plan_id", UNSET)
        current_plan_id: UUID | Unset
        if isinstance(_current_plan_id, Unset):
            current_plan_id = UNSET
        else:
            current_plan_id = UUID(_current_plan_id)

        _new_plan_id = d.pop("new_plan_id", UNSET)
        new_plan_id: UUID | Unset
        if isinstance(_new_plan_id, Unset):
            new_plan_id = UNSET
        else:
            new_plan_id = UUID(_new_plan_id)

        currency = d.pop("currency", UNSET)

        credit_amount = d.pop("credit_amount", UNSET)

        charge_amount = d.pop("charge_amount", UNSET)

        net_amount = d.pop("net_amount", UNSET)

        tax_amount = d.pop("tax_amount", UNSET)

        total_amount = d.pop("total_amount", UNSET)

        _effective_date = d.pop("effective_date", UNSET)
        effective_date: datetime.datetime | Unset
        if isinstance(_effective_date, Unset):
            effective_date = UNSET
        else:
            effective_date = datetime.datetime.fromisoformat(_effective_date)

        next_invoice_amount = d.pop("next_invoice_amount", UNSET)

        is_upgrade = d.pop("is_upgrade", UNSET)

        plan_change_preview = cls(
            subscription_id=subscription_id,
            current_plan_id=current_plan_id,
            new_plan_id=new_plan_id,
            currency=currency,
            credit_amount=credit_amount,
            charge_amount=charge_amount,
            net_amount=net_amount,
            tax_amount=tax_amount,
            total_amount=total_amount,
            effective_date=effective_date,
            next_invoice_amount=next_invoice_amount,
            is_upgrade=is_upgrade,
        )

        plan_change_preview.additional_properties = d
        return plan_change_preview

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
