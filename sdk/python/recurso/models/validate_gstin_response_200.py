from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ValidateGSTINResponse200")


@_attrs_define
class ValidateGSTINResponse200:
    """
    Attributes:
        valid (bool | Unset):
        state_code (str | Unset): Present when valid.
        state_name (str | Unset): Present when valid.
        pan (str | Unset): Present when valid.
        message (str | Unset):
    """

    valid: bool | Unset = UNSET
    state_code: str | Unset = UNSET
    state_name: str | Unset = UNSET
    pan: str | Unset = UNSET
    message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        valid = self.valid

        state_code = self.state_code

        state_name = self.state_name

        pan = self.pan

        message = self.message

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if valid is not UNSET:
            field_dict["valid"] = valid
        if state_code is not UNSET:
            field_dict["state_code"] = state_code
        if state_name is not UNSET:
            field_dict["state_name"] = state_name
        if pan is not UNSET:
            field_dict["pan"] = pan
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        valid = d.pop("valid", UNSET)

        state_code = d.pop("state_code", UNSET)

        state_name = d.pop("state_name", UNSET)

        pan = d.pop("pan", UNSET)

        message = d.pop("message", UNSET)

        validate_gstin_response_200 = cls(
            valid=valid,
            state_code=state_code,
            state_name=state_name,
            pan=pan,
            message=message,
        )

        validate_gstin_response_200.additional_properties = d
        return validate_gstin_response_200

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
