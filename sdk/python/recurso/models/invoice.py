from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.invoice_status import InvoiceStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="Invoice")


@_attrs_define
class Invoice:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        subscription_id (None | Unset | UUID):
        customer_id (UUID | Unset):
        invoice_number (str | Unset):
        billing_reason (str | Unset):
        amount_due (int | Unset):
        amount_paid (int | Unset):
        currency (str | Unset):
        subtotal (int | Unset):
        tax_amount (int | Unset):
        total (int | Unset):
        igst_amount (int | Unset):
        cgst_amount (int | Unset):
        sgst_amount (int | Unset):
        hsn_code (str | Unset):
        irn (str | Unset): Invoice Reference Number issued by the IRP (Indian e-invoicing).
        ack_no (str | Unset):
        signed_qr_code (str | Unset):
        e_invoice_status (str | Unset):
        ack_date (str | Unset):
        e_invoice_retry_count (int | Unset):
        e_invoice_next_retry_at (datetime.datetime | None | Unset):
        e_invoice_error_message (str | Unset):
        tds_amount (int | Unset):
        status (InvoiceStatus | Unset):
        created_at (datetime.datetime | Unset):
        due_date (datetime.datetime | Unset):
        paid_at (datetime.datetime | Unset):
        payment_terms (str | Unset):
        exchange_rate (float | Unset):
        base_currency_total (int | Unset):
        base_currency (str | Unset):
        next_retry_at (datetime.datetime | Unset):
        retry_count (int | Unset):
        payment_wall_active (bool | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    subscription_id: None | Unset | UUID = UNSET
    customer_id: UUID | Unset = UNSET
    invoice_number: str | Unset = UNSET
    billing_reason: str | Unset = UNSET
    amount_due: int | Unset = UNSET
    amount_paid: int | Unset = UNSET
    currency: str | Unset = UNSET
    subtotal: int | Unset = UNSET
    tax_amount: int | Unset = UNSET
    total: int | Unset = UNSET
    igst_amount: int | Unset = UNSET
    cgst_amount: int | Unset = UNSET
    sgst_amount: int | Unset = UNSET
    hsn_code: str | Unset = UNSET
    irn: str | Unset = UNSET
    ack_no: str | Unset = UNSET
    signed_qr_code: str | Unset = UNSET
    e_invoice_status: str | Unset = UNSET
    ack_date: str | Unset = UNSET
    e_invoice_retry_count: int | Unset = UNSET
    e_invoice_next_retry_at: datetime.datetime | None | Unset = UNSET
    e_invoice_error_message: str | Unset = UNSET
    tds_amount: int | Unset = UNSET
    status: InvoiceStatus | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    due_date: datetime.datetime | Unset = UNSET
    paid_at: datetime.datetime | Unset = UNSET
    payment_terms: str | Unset = UNSET
    exchange_rate: float | Unset = UNSET
    base_currency_total: int | Unset = UNSET
    base_currency: str | Unset = UNSET
    next_retry_at: datetime.datetime | Unset = UNSET
    retry_count: int | Unset = UNSET
    payment_wall_active: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        subscription_id: None | str | Unset
        if isinstance(self.subscription_id, Unset):
            subscription_id = UNSET
        elif isinstance(self.subscription_id, UUID):
            subscription_id = str(self.subscription_id)
        else:
            subscription_id = self.subscription_id

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        invoice_number = self.invoice_number

        billing_reason = self.billing_reason

        amount_due = self.amount_due

        amount_paid = self.amount_paid

        currency = self.currency

        subtotal = self.subtotal

        tax_amount = self.tax_amount

        total = self.total

        igst_amount = self.igst_amount

        cgst_amount = self.cgst_amount

        sgst_amount = self.sgst_amount

        hsn_code = self.hsn_code

        irn = self.irn

        ack_no = self.ack_no

        signed_qr_code = self.signed_qr_code

        e_invoice_status = self.e_invoice_status

        ack_date = self.ack_date

        e_invoice_retry_count = self.e_invoice_retry_count

        e_invoice_next_retry_at: None | str | Unset
        if isinstance(self.e_invoice_next_retry_at, Unset):
            e_invoice_next_retry_at = UNSET
        elif isinstance(self.e_invoice_next_retry_at, datetime.datetime):
            e_invoice_next_retry_at = self.e_invoice_next_retry_at.isoformat()
        else:
            e_invoice_next_retry_at = self.e_invoice_next_retry_at

        e_invoice_error_message = self.e_invoice_error_message

        tds_amount = self.tds_amount

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        due_date: str | Unset = UNSET
        if not isinstance(self.due_date, Unset):
            due_date = self.due_date.isoformat()

        paid_at: str | Unset = UNSET
        if not isinstance(self.paid_at, Unset):
            paid_at = self.paid_at.isoformat()

        payment_terms = self.payment_terms

        exchange_rate = self.exchange_rate

        base_currency_total = self.base_currency_total

        base_currency = self.base_currency

        next_retry_at: str | Unset = UNSET
        if not isinstance(self.next_retry_at, Unset):
            next_retry_at = self.next_retry_at.isoformat()

        retry_count = self.retry_count

        payment_wall_active = self.payment_wall_active

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if subscription_id is not UNSET:
            field_dict["subscription_id"] = subscription_id
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if invoice_number is not UNSET:
            field_dict["invoice_number"] = invoice_number
        if billing_reason is not UNSET:
            field_dict["billing_reason"] = billing_reason
        if amount_due is not UNSET:
            field_dict["amount_due"] = amount_due
        if amount_paid is not UNSET:
            field_dict["amount_paid"] = amount_paid
        if currency is not UNSET:
            field_dict["currency"] = currency
        if subtotal is not UNSET:
            field_dict["subtotal"] = subtotal
        if tax_amount is not UNSET:
            field_dict["tax_amount"] = tax_amount
        if total is not UNSET:
            field_dict["total"] = total
        if igst_amount is not UNSET:
            field_dict["igst_amount"] = igst_amount
        if cgst_amount is not UNSET:
            field_dict["cgst_amount"] = cgst_amount
        if sgst_amount is not UNSET:
            field_dict["sgst_amount"] = sgst_amount
        if hsn_code is not UNSET:
            field_dict["hsn_code"] = hsn_code
        if irn is not UNSET:
            field_dict["irn"] = irn
        if ack_no is not UNSET:
            field_dict["ack_no"] = ack_no
        if signed_qr_code is not UNSET:
            field_dict["signed_qr_code"] = signed_qr_code
        if e_invoice_status is not UNSET:
            field_dict["e_invoice_status"] = e_invoice_status
        if ack_date is not UNSET:
            field_dict["ack_date"] = ack_date
        if e_invoice_retry_count is not UNSET:
            field_dict["e_invoice_retry_count"] = e_invoice_retry_count
        if e_invoice_next_retry_at is not UNSET:
            field_dict["e_invoice_next_retry_at"] = e_invoice_next_retry_at
        if e_invoice_error_message is not UNSET:
            field_dict["e_invoice_error_message"] = e_invoice_error_message
        if tds_amount is not UNSET:
            field_dict["tds_amount"] = tds_amount
        if status is not UNSET:
            field_dict["status"] = status
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if due_date is not UNSET:
            field_dict["due_date"] = due_date
        if paid_at is not UNSET:
            field_dict["paid_at"] = paid_at
        if payment_terms is not UNSET:
            field_dict["payment_terms"] = payment_terms
        if exchange_rate is not UNSET:
            field_dict["exchange_rate"] = exchange_rate
        if base_currency_total is not UNSET:
            field_dict["base_currency_total"] = base_currency_total
        if base_currency is not UNSET:
            field_dict["base_currency"] = base_currency
        if next_retry_at is not UNSET:
            field_dict["next_retry_at"] = next_retry_at
        if retry_count is not UNSET:
            field_dict["retry_count"] = retry_count
        if payment_wall_active is not UNSET:
            field_dict["payment_wall_active"] = payment_wall_active

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

        def _parse_subscription_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                subscription_id_type_0 = UUID(data)

                return subscription_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        subscription_id = _parse_subscription_id(d.pop("subscription_id", UNSET))

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        invoice_number = d.pop("invoice_number", UNSET)

        billing_reason = d.pop("billing_reason", UNSET)

        amount_due = d.pop("amount_due", UNSET)

        amount_paid = d.pop("amount_paid", UNSET)

        currency = d.pop("currency", UNSET)

        subtotal = d.pop("subtotal", UNSET)

        tax_amount = d.pop("tax_amount", UNSET)

        total = d.pop("total", UNSET)

        igst_amount = d.pop("igst_amount", UNSET)

        cgst_amount = d.pop("cgst_amount", UNSET)

        sgst_amount = d.pop("sgst_amount", UNSET)

        hsn_code = d.pop("hsn_code", UNSET)

        irn = d.pop("irn", UNSET)

        ack_no = d.pop("ack_no", UNSET)

        signed_qr_code = d.pop("signed_qr_code", UNSET)

        e_invoice_status = d.pop("e_invoice_status", UNSET)

        ack_date = d.pop("ack_date", UNSET)

        e_invoice_retry_count = d.pop("e_invoice_retry_count", UNSET)

        def _parse_e_invoice_next_retry_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                e_invoice_next_retry_at_type_0 = datetime.datetime.fromisoformat(data)

                return e_invoice_next_retry_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        e_invoice_next_retry_at = _parse_e_invoice_next_retry_at(d.pop("e_invoice_next_retry_at", UNSET))

        e_invoice_error_message = d.pop("e_invoice_error_message", UNSET)

        tds_amount = d.pop("tds_amount", UNSET)

        _status = d.pop("status", UNSET)
        status: InvoiceStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = InvoiceStatus(_status)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        _due_date = d.pop("due_date", UNSET)
        due_date: datetime.datetime | Unset
        if isinstance(_due_date, Unset):
            due_date = UNSET
        else:
            due_date = datetime.datetime.fromisoformat(_due_date)

        _paid_at = d.pop("paid_at", UNSET)
        paid_at: datetime.datetime | Unset
        if isinstance(_paid_at, Unset):
            paid_at = UNSET
        else:
            paid_at = datetime.datetime.fromisoformat(_paid_at)

        payment_terms = d.pop("payment_terms", UNSET)

        exchange_rate = d.pop("exchange_rate", UNSET)

        base_currency_total = d.pop("base_currency_total", UNSET)

        base_currency = d.pop("base_currency", UNSET)

        _next_retry_at = d.pop("next_retry_at", UNSET)
        next_retry_at: datetime.datetime | Unset
        if isinstance(_next_retry_at, Unset):
            next_retry_at = UNSET
        else:
            next_retry_at = datetime.datetime.fromisoformat(_next_retry_at)

        retry_count = d.pop("retry_count", UNSET)

        payment_wall_active = d.pop("payment_wall_active", UNSET)

        invoice = cls(
            id=id,
            tenant_id=tenant_id,
            subscription_id=subscription_id,
            customer_id=customer_id,
            invoice_number=invoice_number,
            billing_reason=billing_reason,
            amount_due=amount_due,
            amount_paid=amount_paid,
            currency=currency,
            subtotal=subtotal,
            tax_amount=tax_amount,
            total=total,
            igst_amount=igst_amount,
            cgst_amount=cgst_amount,
            sgst_amount=sgst_amount,
            hsn_code=hsn_code,
            irn=irn,
            ack_no=ack_no,
            signed_qr_code=signed_qr_code,
            e_invoice_status=e_invoice_status,
            ack_date=ack_date,
            e_invoice_retry_count=e_invoice_retry_count,
            e_invoice_next_retry_at=e_invoice_next_retry_at,
            e_invoice_error_message=e_invoice_error_message,
            tds_amount=tds_amount,
            status=status,
            created_at=created_at,
            due_date=due_date,
            paid_at=paid_at,
            payment_terms=payment_terms,
            exchange_rate=exchange_rate,
            base_currency_total=base_currency_total,
            base_currency=base_currency,
            next_retry_at=next_retry_at,
            retry_count=retry_count,
            payment_wall_active=payment_wall_active,
        )

        invoice.additional_properties = d
        return invoice

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
