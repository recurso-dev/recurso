from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.credit_note_status import CreditNoteStatus
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.customer import Customer


T = TypeVar("T", bound="CreditNote")


@_attrs_define
class CreditNote:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        invoice_id (None | Unset | UUID):
        reference (None | str | Unset):
        amount (int | Unset):
        balance (int | Unset): Remaining unapplied credit.
        currency (str | Unset):
        status (CreditNoteStatus | Unset):
        reason (str | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
        customer (Customer | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    invoice_id: None | Unset | UUID = UNSET
    reference: None | str | Unset = UNSET
    amount: int | Unset = UNSET
    balance: int | Unset = UNSET
    currency: str | Unset = UNSET
    status: CreditNoteStatus | Unset = UNSET
    reason: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    customer: Customer | Unset = UNSET
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

        reference: None | str | Unset
        if isinstance(self.reference, Unset):
            reference = UNSET
        else:
            reference = self.reference

        amount = self.amount

        balance = self.balance

        currency = self.currency

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        reason = self.reason

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        customer: dict[str, Any] | Unset = UNSET
        if not isinstance(self.customer, Unset):
            customer = self.customer.to_dict()

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
        if reference is not UNSET:
            field_dict["reference"] = reference
        if amount is not UNSET:
            field_dict["amount"] = amount
        if balance is not UNSET:
            field_dict["balance"] = balance
        if currency is not UNSET:
            field_dict["currency"] = currency
        if status is not UNSET:
            field_dict["status"] = status
        if reason is not UNSET:
            field_dict["reason"] = reason
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at
        if customer is not UNSET:
            field_dict["customer"] = customer

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.customer import Customer

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

        def _parse_reference(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        reference = _parse_reference(d.pop("reference", UNSET))

        amount = d.pop("amount", UNSET)

        balance = d.pop("balance", UNSET)

        currency = d.pop("currency", UNSET)

        _status = d.pop("status", UNSET)
        status: CreditNoteStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = CreditNoteStatus(_status)

        reason = d.pop("reason", UNSET)

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

        _customer = d.pop("customer", UNSET)
        customer: Customer | Unset
        if isinstance(_customer, Unset):
            customer = UNSET
        else:
            customer = Customer.from_dict(_customer)

        credit_note = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            invoice_id=invoice_id,
            reference=reference,
            amount=amount,
            balance=balance,
            currency=currency,
            status=status,
            reason=reason,
            created_at=created_at,
            updated_at=updated_at,
            customer=customer,
        )

        credit_note.additional_properties = d
        return credit_note

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
