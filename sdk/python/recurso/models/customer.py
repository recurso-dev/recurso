from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.billing_address import BillingAddress
    from ..models.customer_risk_factors_type_0 import CustomerRiskFactorsType0


T = TypeVar("T", bound="Customer")


@_attrs_define
class Customer:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        email (str | Unset):
        name (None | str | Unset):
        phone (str | Unset):
        tax_id (None | str | Unset):
        billing_address (BillingAddress | Unset):
        ledger_account_id (UUID | Unset):
        gstin (None | str | Unset):
        tax_type (str | Unset):
        place_of_supply (None | str | Unset):
        referral_code (None | str | Unset):
        risk_score (int | Unset):
        risk_factors (CustomerRiskFactorsType0 | None | Unset):
        card_brand (str | Unset):
        card_last4 (str | Unset):
        card_exp_month (int | Unset):
        card_exp_year (int | Unset):
        card_token_id (str | Unset):
        card_fingerprint (str | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    email: str | Unset = UNSET
    name: None | str | Unset = UNSET
    phone: str | Unset = UNSET
    tax_id: None | str | Unset = UNSET
    billing_address: BillingAddress | Unset = UNSET
    ledger_account_id: UUID | Unset = UNSET
    gstin: None | str | Unset = UNSET
    tax_type: str | Unset = UNSET
    place_of_supply: None | str | Unset = UNSET
    referral_code: None | str | Unset = UNSET
    risk_score: int | Unset = UNSET
    risk_factors: CustomerRiskFactorsType0 | None | Unset = UNSET
    card_brand: str | Unset = UNSET
    card_last4: str | Unset = UNSET
    card_exp_month: int | Unset = UNSET
    card_exp_year: int | Unset = UNSET
    card_token_id: str | Unset = UNSET
    card_fingerprint: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.customer_risk_factors_type_0 import CustomerRiskFactorsType0

        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        email = self.email

        name: None | str | Unset
        if isinstance(self.name, Unset):
            name = UNSET
        else:
            name = self.name

        phone = self.phone

        tax_id: None | str | Unset
        if isinstance(self.tax_id, Unset):
            tax_id = UNSET
        else:
            tax_id = self.tax_id

        billing_address: dict[str, Any] | Unset = UNSET
        if not isinstance(self.billing_address, Unset):
            billing_address = self.billing_address.to_dict()

        ledger_account_id: str | Unset = UNSET
        if not isinstance(self.ledger_account_id, Unset):
            ledger_account_id = str(self.ledger_account_id)

        gstin: None | str | Unset
        if isinstance(self.gstin, Unset):
            gstin = UNSET
        else:
            gstin = self.gstin

        tax_type = self.tax_type

        place_of_supply: None | str | Unset
        if isinstance(self.place_of_supply, Unset):
            place_of_supply = UNSET
        else:
            place_of_supply = self.place_of_supply

        referral_code: None | str | Unset
        if isinstance(self.referral_code, Unset):
            referral_code = UNSET
        else:
            referral_code = self.referral_code

        risk_score = self.risk_score

        risk_factors: dict[str, Any] | None | Unset
        if isinstance(self.risk_factors, Unset):
            risk_factors = UNSET
        elif isinstance(self.risk_factors, CustomerRiskFactorsType0):
            risk_factors = self.risk_factors.to_dict()
        else:
            risk_factors = self.risk_factors

        card_brand = self.card_brand

        card_last4 = self.card_last4

        card_exp_month = self.card_exp_month

        card_exp_year = self.card_exp_year

        card_token_id = self.card_token_id

        card_fingerprint = self.card_fingerprint

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
        if email is not UNSET:
            field_dict["email"] = email
        if name is not UNSET:
            field_dict["name"] = name
        if phone is not UNSET:
            field_dict["phone"] = phone
        if tax_id is not UNSET:
            field_dict["tax_id"] = tax_id
        if billing_address is not UNSET:
            field_dict["billing_address"] = billing_address
        if ledger_account_id is not UNSET:
            field_dict["ledger_account_id"] = ledger_account_id
        if gstin is not UNSET:
            field_dict["gstin"] = gstin
        if tax_type is not UNSET:
            field_dict["tax_type"] = tax_type
        if place_of_supply is not UNSET:
            field_dict["place_of_supply"] = place_of_supply
        if referral_code is not UNSET:
            field_dict["referral_code"] = referral_code
        if risk_score is not UNSET:
            field_dict["risk_score"] = risk_score
        if risk_factors is not UNSET:
            field_dict["risk_factors"] = risk_factors
        if card_brand is not UNSET:
            field_dict["card_brand"] = card_brand
        if card_last4 is not UNSET:
            field_dict["card_last4"] = card_last4
        if card_exp_month is not UNSET:
            field_dict["card_exp_month"] = card_exp_month
        if card_exp_year is not UNSET:
            field_dict["card_exp_year"] = card_exp_year
        if card_token_id is not UNSET:
            field_dict["card_token_id"] = card_token_id
        if card_fingerprint is not UNSET:
            field_dict["card_fingerprint"] = card_fingerprint
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.billing_address import BillingAddress
        from ..models.customer_risk_factors_type_0 import CustomerRiskFactorsType0

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

        email = d.pop("email", UNSET)

        def _parse_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        name = _parse_name(d.pop("name", UNSET))

        phone = d.pop("phone", UNSET)

        def _parse_tax_id(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        tax_id = _parse_tax_id(d.pop("tax_id", UNSET))

        _billing_address = d.pop("billing_address", UNSET)
        billing_address: BillingAddress | Unset
        if isinstance(_billing_address, Unset):
            billing_address = UNSET
        else:
            billing_address = BillingAddress.from_dict(_billing_address)

        _ledger_account_id = d.pop("ledger_account_id", UNSET)
        ledger_account_id: UUID | Unset
        if isinstance(_ledger_account_id, Unset):
            ledger_account_id = UNSET
        else:
            ledger_account_id = UUID(_ledger_account_id)

        def _parse_gstin(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        gstin = _parse_gstin(d.pop("gstin", UNSET))

        tax_type = d.pop("tax_type", UNSET)

        def _parse_place_of_supply(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        place_of_supply = _parse_place_of_supply(d.pop("place_of_supply", UNSET))

        def _parse_referral_code(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        referral_code = _parse_referral_code(d.pop("referral_code", UNSET))

        risk_score = d.pop("risk_score", UNSET)

        def _parse_risk_factors(data: object) -> CustomerRiskFactorsType0 | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                risk_factors_type_0 = CustomerRiskFactorsType0.from_dict(data)

                return risk_factors_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(CustomerRiskFactorsType0 | None | Unset, data)

        risk_factors = _parse_risk_factors(d.pop("risk_factors", UNSET))

        card_brand = d.pop("card_brand", UNSET)

        card_last4 = d.pop("card_last4", UNSET)

        card_exp_month = d.pop("card_exp_month", UNSET)

        card_exp_year = d.pop("card_exp_year", UNSET)

        card_token_id = d.pop("card_token_id", UNSET)

        card_fingerprint = d.pop("card_fingerprint", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        customer = cls(
            id=id,
            tenant_id=tenant_id,
            email=email,
            name=name,
            phone=phone,
            tax_id=tax_id,
            billing_address=billing_address,
            ledger_account_id=ledger_account_id,
            gstin=gstin,
            tax_type=tax_type,
            place_of_supply=place_of_supply,
            referral_code=referral_code,
            risk_score=risk_score,
            risk_factors=risk_factors,
            card_brand=card_brand,
            card_last4=card_last4,
            card_exp_month=card_exp_month,
            card_exp_year=card_exp_year,
            card_token_id=card_token_id,
            card_fingerprint=card_fingerprint,
            created_at=created_at,
        )

        customer.additional_properties = d
        return customer

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
