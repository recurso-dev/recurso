from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="CancelEInvoiceBody")


@_attrs_define
class CancelEInvoiceBody:
    """
    Attributes:
        cancel_code (int): IRP cancellation code (e.g. 1 = duplicate, 3 = data entry mistake).
        reason (str):
    """

    cancel_code: int
    reason: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cancel_code = self.cancel_code

        reason = self.reason

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cancel_code": cancel_code,
                "reason": reason,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cancel_code = d.pop("cancel_code")

        reason = d.pop("reason")

        cancel_e_invoice_body = cls(
            cancel_code=cancel_code,
            reason=reason,
        )

        cancel_e_invoice_body.additional_properties = d
        return cancel_e_invoice_body

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
