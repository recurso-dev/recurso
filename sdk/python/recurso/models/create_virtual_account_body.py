from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateVirtualAccountBody")


@_attrs_define
class CreateVirtualAccountBody:
    """
    Attributes:
        customer_id (UUID):
        amount (int): Expected amount in the lowest currency unit.
        invoice_id (UUID | Unset):
    """

    customer_id: UUID
    amount: int
    invoice_id: UUID | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        customer_id = str(self.customer_id)

        amount = self.amount

        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "customer_id": customer_id,
                "amount": amount,
            }
        )
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        customer_id = UUID(d.pop("customer_id"))

        amount = d.pop("amount")

        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        create_virtual_account_body = cls(
            customer_id=customer_id,
            amount=amount,
            invoice_id=invoice_id,
        )

        create_virtual_account_body.additional_properties = d
        return create_virtual_account_body

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
