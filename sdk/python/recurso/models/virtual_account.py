from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="VirtualAccount")


@_attrs_define
class VirtualAccount:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        invoice_id (None | Unset | UUID):
        account_number (str | Unset):
        ifsc_code (str | Unset):
        bank_name (str | Unset):
        beneficiary_name (str | Unset):
        razorpay_va_id (str | Unset):
        status (str | Unset):
        amount_expected (int | Unset):
        amount_received (int | Unset):
        closed_at (datetime.datetime | None | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    invoice_id: None | Unset | UUID = UNSET
    account_number: str | Unset = UNSET
    ifsc_code: str | Unset = UNSET
    bank_name: str | Unset = UNSET
    beneficiary_name: str | Unset = UNSET
    razorpay_va_id: str | Unset = UNSET
    status: str | Unset = UNSET
    amount_expected: int | Unset = UNSET
    amount_received: int | Unset = UNSET
    closed_at: datetime.datetime | None | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
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

        invoice_id: None | str | Unset
        if isinstance(self.invoice_id, Unset):
            invoice_id = UNSET
        elif isinstance(self.invoice_id, UUID):
            invoice_id = str(self.invoice_id)
        else:
            invoice_id = self.invoice_id

        account_number = self.account_number

        ifsc_code = self.ifsc_code

        bank_name = self.bank_name

        beneficiary_name = self.beneficiary_name

        razorpay_va_id = self.razorpay_va_id

        status = self.status

        amount_expected = self.amount_expected

        amount_received = self.amount_received

        closed_at: None | str | Unset
        if isinstance(self.closed_at, Unset):
            closed_at = UNSET
        elif isinstance(self.closed_at, datetime.datetime):
            closed_at = self.closed_at.isoformat()
        else:
            closed_at = self.closed_at

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
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if account_number is not UNSET:
            field_dict["account_number"] = account_number
        if ifsc_code is not UNSET:
            field_dict["ifsc_code"] = ifsc_code
        if bank_name is not UNSET:
            field_dict["bank_name"] = bank_name
        if beneficiary_name is not UNSET:
            field_dict["beneficiary_name"] = beneficiary_name
        if razorpay_va_id is not UNSET:
            field_dict["razorpay_va_id"] = razorpay_va_id
        if status is not UNSET:
            field_dict["status"] = status
        if amount_expected is not UNSET:
            field_dict["amount_expected"] = amount_expected
        if amount_received is not UNSET:
            field_dict["amount_received"] = amount_received
        if closed_at is not UNSET:
            field_dict["closed_at"] = closed_at
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

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        def _parse_invoice_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                invoice_id_type_0 = UUID(data)

                return invoice_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        invoice_id = _parse_invoice_id(d.pop("invoice_id", UNSET))

        account_number = d.pop("account_number", UNSET)

        ifsc_code = d.pop("ifsc_code", UNSET)

        bank_name = d.pop("bank_name", UNSET)

        beneficiary_name = d.pop("beneficiary_name", UNSET)

        razorpay_va_id = d.pop("razorpay_va_id", UNSET)

        status = d.pop("status", UNSET)

        amount_expected = d.pop("amount_expected", UNSET)

        amount_received = d.pop("amount_received", UNSET)

        def _parse_closed_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                closed_at_type_0 = datetime.datetime.fromisoformat(data)

                return closed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        closed_at = _parse_closed_at(d.pop("closed_at", UNSET))

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        virtual_account = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            invoice_id=invoice_id,
            account_number=account_number,
            ifsc_code=ifsc_code,
            bank_name=bank_name,
            beneficiary_name=beneficiary_name,
            razorpay_va_id=razorpay_va_id,
            status=status,
            amount_expected=amount_expected,
            amount_received=amount_received,
            closed_at=closed_at,
            created_at=created_at,
        )

        virtual_account.additional_properties = d
        return virtual_account

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
