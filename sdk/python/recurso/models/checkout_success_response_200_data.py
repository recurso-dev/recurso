from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.checkout_success_response_200_data_status import CheckoutSuccessResponse200DataStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="CheckoutSuccessResponse200Data")


@_attrs_define
class CheckoutSuccessResponse200Data:
    """
    Attributes:
        status (CheckoutSuccessResponse200DataStatus | Unset):
        payment_status (str | Unset): Raw gateway intent status (present when a payment_intent was inspected).
        invoice_id (UUID | Unset):
        invoice_number (str | Unset):
    """

    status: CheckoutSuccessResponse200DataStatus | Unset = UNSET
    payment_status: str | Unset = UNSET
    invoice_id: UUID | Unset = UNSET
    invoice_number: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        payment_status = self.payment_status

        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        invoice_number = self.invoice_number

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if status is not UNSET:
            field_dict["status"] = status
        if payment_status is not UNSET:
            field_dict["payment_status"] = payment_status
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if invoice_number is not UNSET:
            field_dict["invoice_number"] = invoice_number

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _status = d.pop("status", UNSET)
        status: CheckoutSuccessResponse200DataStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = CheckoutSuccessResponse200DataStatus(_status)

        payment_status = d.pop("payment_status", UNSET)

        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        invoice_number = d.pop("invoice_number", UNSET)

        checkout_success_response_200_data = cls(
            status=status,
            payment_status=payment_status,
            invoice_id=invoice_id,
            invoice_number=invoice_number,
        )

        checkout_success_response_200_data.additional_properties = d
        return checkout_success_response_200_data

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
