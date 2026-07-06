from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="EInvoiceStatus")


@_attrs_define
class EInvoiceStatus:
    """
    Attributes:
        invoice_id (UUID | Unset):
        invoice_number (str | Unset):
        e_invoice_status (str | Unset):
        irn (str | Unset):
        ack_no (str | Unset):
        ack_date (str | Unset):
        signed_qr_code (str | Unset):
        retry_count (int | Unset):
        next_retry_at (datetime.datetime | None | Unset):
        error_message (str | Unset):
    """

    invoice_id: UUID | Unset = UNSET
    invoice_number: str | Unset = UNSET
    e_invoice_status: str | Unset = UNSET
    irn: str | Unset = UNSET
    ack_no: str | Unset = UNSET
    ack_date: str | Unset = UNSET
    signed_qr_code: str | Unset = UNSET
    retry_count: int | Unset = UNSET
    next_retry_at: datetime.datetime | None | Unset = UNSET
    error_message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        invoice_number = self.invoice_number

        e_invoice_status = self.e_invoice_status

        irn = self.irn

        ack_no = self.ack_no

        ack_date = self.ack_date

        signed_qr_code = self.signed_qr_code

        retry_count = self.retry_count

        next_retry_at: None | str | Unset
        if isinstance(self.next_retry_at, Unset):
            next_retry_at = UNSET
        elif isinstance(self.next_retry_at, datetime.datetime):
            next_retry_at = self.next_retry_at.isoformat()
        else:
            next_retry_at = self.next_retry_at

        error_message = self.error_message

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if invoice_number is not UNSET:
            field_dict["invoice_number"] = invoice_number
        if e_invoice_status is not UNSET:
            field_dict["e_invoice_status"] = e_invoice_status
        if irn is not UNSET:
            field_dict["irn"] = irn
        if ack_no is not UNSET:
            field_dict["ack_no"] = ack_no
        if ack_date is not UNSET:
            field_dict["ack_date"] = ack_date
        if signed_qr_code is not UNSET:
            field_dict["signed_qr_code"] = signed_qr_code
        if retry_count is not UNSET:
            field_dict["retry_count"] = retry_count
        if next_retry_at is not UNSET:
            field_dict["next_retry_at"] = next_retry_at
        if error_message is not UNSET:
            field_dict["error_message"] = error_message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        invoice_number = d.pop("invoice_number", UNSET)

        e_invoice_status = d.pop("e_invoice_status", UNSET)

        irn = d.pop("irn", UNSET)

        ack_no = d.pop("ack_no", UNSET)

        ack_date = d.pop("ack_date", UNSET)

        signed_qr_code = d.pop("signed_qr_code", UNSET)

        retry_count = d.pop("retry_count", UNSET)

        def _parse_next_retry_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                next_retry_at_type_0 = datetime.datetime.fromisoformat(data)

                return next_retry_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        next_retry_at = _parse_next_retry_at(d.pop("next_retry_at", UNSET))

        error_message = d.pop("error_message", UNSET)

        e_invoice_status = cls(
            invoice_id=invoice_id,
            invoice_number=invoice_number,
            e_invoice_status=e_invoice_status,
            irn=irn,
            ack_no=ack_no,
            ack_date=ack_date,
            signed_qr_code=signed_qr_code,
            retry_count=retry_count,
            next_retry_at=next_retry_at,
            error_message=error_message,
        )

        e_invoice_status.additional_properties = d
        return e_invoice_status

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
