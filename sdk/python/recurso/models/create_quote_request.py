from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.line_item import LineItem


T = TypeVar("T", bound="CreateQuoteRequest")


@_attrs_define
class CreateQuoteRequest:
    """
    Attributes:
        customer_id (UUID):
        line_items (list[LineItem]):
        currency (str | Unset):
        valid_until (datetime.datetime | None | Unset):
        notes (str | Unset):
        terms (str | Unset):
        tax_amount (int | Unset):
        discount_amount (int | Unset):
    """

    customer_id: UUID
    line_items: list[LineItem]
    currency: str | Unset = UNSET
    valid_until: datetime.datetime | None | Unset = UNSET
    notes: str | Unset = UNSET
    terms: str | Unset = UNSET
    tax_amount: int | Unset = UNSET
    discount_amount: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        customer_id = str(self.customer_id)

        line_items = []
        for line_items_item_data in self.line_items:
            line_items_item = line_items_item_data.to_dict()
            line_items.append(line_items_item)

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

        tax_amount = self.tax_amount

        discount_amount = self.discount_amount

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "customer_id": customer_id,
                "line_items": line_items,
            }
        )
        if currency is not UNSET:
            field_dict["currency"] = currency
        if valid_until is not UNSET:
            field_dict["valid_until"] = valid_until
        if notes is not UNSET:
            field_dict["notes"] = notes
        if terms is not UNSET:
            field_dict["terms"] = terms
        if tax_amount is not UNSET:
            field_dict["tax_amount"] = tax_amount
        if discount_amount is not UNSET:
            field_dict["discount_amount"] = discount_amount

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.line_item import LineItem

        d = dict(src_dict)
        customer_id = UUID(d.pop("customer_id"))

        line_items = []
        _line_items = d.pop("line_items")
        for line_items_item_data in _line_items:
            line_items_item = LineItem.from_dict(line_items_item_data)

            line_items.append(line_items_item)

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

        tax_amount = d.pop("tax_amount", UNSET)

        discount_amount = d.pop("discount_amount", UNSET)

        create_quote_request = cls(
            customer_id=customer_id,
            line_items=line_items,
            currency=currency,
            valid_until=valid_until,
            notes=notes,
            terms=terms,
            tax_amount=tax_amount,
            discount_amount=discount_amount,
        )

        create_quote_request.additional_properties = d
        return create_quote_request

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
