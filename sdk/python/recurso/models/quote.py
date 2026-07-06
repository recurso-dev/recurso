from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.quote_status import QuoteStatus
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.line_item import LineItem


T = TypeVar("T", bound="Quote")


@_attrs_define
class Quote:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        quote_number (str | Unset):
        status (QuoteStatus | Unset):
        line_items (list[LineItem] | Unset):
        subtotal (int | Unset):
        tax_amount (int | Unset):
        discount_amount (int | Unset):
        total (int | Unset):
        currency (str | Unset):
        valid_until (datetime.datetime | None | Unset):
        notes (str | Unset):
        terms (str | Unset):
        invoice_id (None | Unset | UUID): Set once the quote has been converted to an invoice.
        accepted_at (datetime.datetime | None | Unset):
        declined_at (datetime.datetime | None | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    quote_number: str | Unset = UNSET
    status: QuoteStatus | Unset = UNSET
    line_items: list[LineItem] | Unset = UNSET
    subtotal: int | Unset = UNSET
    tax_amount: int | Unset = UNSET
    discount_amount: int | Unset = UNSET
    total: int | Unset = UNSET
    currency: str | Unset = UNSET
    valid_until: datetime.datetime | None | Unset = UNSET
    notes: str | Unset = UNSET
    terms: str | Unset = UNSET
    invoice_id: None | Unset | UUID = UNSET
    accepted_at: datetime.datetime | None | Unset = UNSET
    declined_at: datetime.datetime | None | Unset = UNSET
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

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        quote_number = self.quote_number

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        line_items: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.line_items, Unset):
            line_items = []
            for line_items_item_data in self.line_items:
                line_items_item = line_items_item_data.to_dict()
                line_items.append(line_items_item)

        subtotal = self.subtotal

        tax_amount = self.tax_amount

        discount_amount = self.discount_amount

        total = self.total

        currency = self.currency

        valid_until: None | str | Unset
        if isinstance(self.valid_until, Unset):
            valid_until = UNSET
        elif isinstance(self.valid_until, datetime.datetime):
            valid_until = self.valid_until.isoformat()
        else:
            valid_until = self.valid_until

        notes = self.notes

        terms = self.terms

        invoice_id: None | str | Unset
        if isinstance(self.invoice_id, Unset):
            invoice_id = UNSET
        elif isinstance(self.invoice_id, UUID):
            invoice_id = str(self.invoice_id)
        else:
            invoice_id = self.invoice_id

        accepted_at: None | str | Unset
        if isinstance(self.accepted_at, Unset):
            accepted_at = UNSET
        elif isinstance(self.accepted_at, datetime.datetime):
            accepted_at = self.accepted_at.isoformat()
        else:
            accepted_at = self.accepted_at

        declined_at: None | str | Unset
        if isinstance(self.declined_at, Unset):
            declined_at = UNSET
        elif isinstance(self.declined_at, datetime.datetime):
            declined_at = self.declined_at.isoformat()
        else:
            declined_at = self.declined_at

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
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if quote_number is not UNSET:
            field_dict["quote_number"] = quote_number
        if status is not UNSET:
            field_dict["status"] = status
        if line_items is not UNSET:
            field_dict["line_items"] = line_items
        if subtotal is not UNSET:
            field_dict["subtotal"] = subtotal
        if tax_amount is not UNSET:
            field_dict["tax_amount"] = tax_amount
        if discount_amount is not UNSET:
            field_dict["discount_amount"] = discount_amount
        if total is not UNSET:
            field_dict["total"] = total
        if currency is not UNSET:
            field_dict["currency"] = currency
        if valid_until is not UNSET:
            field_dict["valid_until"] = valid_until
        if notes is not UNSET:
            field_dict["notes"] = notes
        if terms is not UNSET:
            field_dict["terms"] = terms
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if accepted_at is not UNSET:
            field_dict["accepted_at"] = accepted_at
        if declined_at is not UNSET:
            field_dict["declined_at"] = declined_at
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.line_item import LineItem

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

        quote_number = d.pop("quote_number", UNSET)

        _status = d.pop("status", UNSET)
        status: QuoteStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = QuoteStatus(_status)

        _line_items = d.pop("line_items", UNSET)
        line_items: list[LineItem] | Unset = UNSET
        if _line_items is not UNSET:
            line_items = []
            for line_items_item_data in _line_items:
                line_items_item = LineItem.from_dict(line_items_item_data)

                line_items.append(line_items_item)

        subtotal = d.pop("subtotal", UNSET)

        tax_amount = d.pop("tax_amount", UNSET)

        discount_amount = d.pop("discount_amount", UNSET)

        total = d.pop("total", UNSET)

        currency = d.pop("currency", UNSET)

        def _parse_valid_until(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                valid_until_type_0 = datetime.datetime.fromisoformat(data)

                return valid_until_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        valid_until = _parse_valid_until(d.pop("valid_until", UNSET))

        notes = d.pop("notes", UNSET)

        terms = d.pop("terms", UNSET)

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

        def _parse_accepted_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                accepted_at_type_0 = datetime.datetime.fromisoformat(data)

                return accepted_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        accepted_at = _parse_accepted_at(d.pop("accepted_at", UNSET))

        def _parse_declined_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                declined_at_type_0 = datetime.datetime.fromisoformat(data)

                return declined_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        declined_at = _parse_declined_at(d.pop("declined_at", UNSET))

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

        quote = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            quote_number=quote_number,
            status=status,
            line_items=line_items,
            subtotal=subtotal,
            tax_amount=tax_amount,
            discount_amount=discount_amount,
            total=total,
            currency=currency,
            valid_until=valid_until,
            notes=notes,
            terms=terms,
            invoice_id=invoice_id,
            accepted_at=accepted_at,
            declined_at=declined_at,
            created_at=created_at,
            updated_at=updated_at,
        )

        quote.additional_properties = d
        return quote

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
