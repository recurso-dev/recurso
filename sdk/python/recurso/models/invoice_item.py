from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="InvoiceItem")


@_attrs_define
class InvoiceItem:
    """
    Attributes:
        id (UUID | Unset):
        invoice_id (UUID | Unset):
        description (str | Unset):
        hsn_code (str | Unset): HSN/SAC code for the line (tenant SAC in Phase 1).
        quantity (int | Unset):
        unit_amount (int | Unset): Per-unit amount in the invoice's lowest currency unit.
        amount (int | Unset): Line total (quantity * unit_amount) in lowest currency unit.
        tax_rate (float | Unset): Effective GST rate for the line as a percent (e.g. 18.0).
        cgst_amount (int | Unset):
        sgst_amount (int | Unset):
        igst_amount (int | Unset):
        taxable_amount (int | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    invoice_id: UUID | Unset = UNSET
    description: str | Unset = UNSET
    hsn_code: str | Unset = UNSET
    quantity: int | Unset = UNSET
    unit_amount: int | Unset = UNSET
    amount: int | Unset = UNSET
    tax_rate: float | Unset = UNSET
    cgst_amount: int | Unset = UNSET
    sgst_amount: int | Unset = UNSET
    igst_amount: int | Unset = UNSET
    taxable_amount: int | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        description = self.description

        hsn_code = self.hsn_code

        quantity = self.quantity

        unit_amount = self.unit_amount

        amount = self.amount

        tax_rate = self.tax_rate

        cgst_amount = self.cgst_amount

        sgst_amount = self.sgst_amount

        igst_amount = self.igst_amount

        taxable_amount = self.taxable_amount

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if description is not UNSET:
            field_dict["description"] = description
        if hsn_code is not UNSET:
            field_dict["hsn_code"] = hsn_code
        if quantity is not UNSET:
            field_dict["quantity"] = quantity
        if unit_amount is not UNSET:
            field_dict["unit_amount"] = unit_amount
        if amount is not UNSET:
            field_dict["amount"] = amount
        if tax_rate is not UNSET:
            field_dict["tax_rate"] = tax_rate
        if cgst_amount is not UNSET:
            field_dict["cgst_amount"] = cgst_amount
        if sgst_amount is not UNSET:
            field_dict["sgst_amount"] = sgst_amount
        if igst_amount is not UNSET:
            field_dict["igst_amount"] = igst_amount
        if taxable_amount is not UNSET:
            field_dict["taxable_amount"] = taxable_amount
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

        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        description = d.pop("description", UNSET)

        hsn_code = d.pop("hsn_code", UNSET)

        quantity = d.pop("quantity", UNSET)

        unit_amount = d.pop("unit_amount", UNSET)

        amount = d.pop("amount", UNSET)

        tax_rate = d.pop("tax_rate", UNSET)

        cgst_amount = d.pop("cgst_amount", UNSET)

        sgst_amount = d.pop("sgst_amount", UNSET)

        igst_amount = d.pop("igst_amount", UNSET)

        taxable_amount = d.pop("taxable_amount", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        invoice_item = cls(
            id=id,
            invoice_id=invoice_id,
            description=description,
            hsn_code=hsn_code,
            quantity=quantity,
            unit_amount=unit_amount,
            amount=amount,
            tax_rate=tax_rate,
            cgst_amount=cgst_amount,
            sgst_amount=sgst_amount,
            igst_amount=igst_amount,
            taxable_amount=taxable_amount,
            created_at=created_at,
        )

        invoice_item.additional_properties = d
        return invoice_item

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
