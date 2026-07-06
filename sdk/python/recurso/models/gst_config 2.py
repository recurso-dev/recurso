from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="GSTConfig")


@_attrs_define
class GSTConfig:
    """
    Attributes:
        gstin (str | Unset):
        state_code (str | Unset): Two-digit Indian state code (derived from the GSTIN when omitted).
        state_name (str | Unset):
        sac_code (str | Unset):
        gst_rate (float | Unset):
        pan (str | Unset):
        legal_name (str | Unset):
        trade_name (str | Unset):
        address (str | Unset):
        has_lut (bool | Unset): Letter of Undertaking on file (zero-rated exports).
    """

    gstin: str | Unset = UNSET
    state_code: str | Unset = UNSET
    state_name: str | Unset = UNSET
    sac_code: str | Unset = UNSET
    gst_rate: float | Unset = UNSET
    pan: str | Unset = UNSET
    legal_name: str | Unset = UNSET
    trade_name: str | Unset = UNSET
    address: str | Unset = UNSET
    has_lut: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        gstin = self.gstin

        state_code = self.state_code

        state_name = self.state_name

        sac_code = self.sac_code

        gst_rate = self.gst_rate

        pan = self.pan

        legal_name = self.legal_name

        trade_name = self.trade_name

        address = self.address

        has_lut = self.has_lut

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if gstin is not UNSET:
            field_dict["gstin"] = gstin
        if state_code is not UNSET:
            field_dict["state_code"] = state_code
        if state_name is not UNSET:
            field_dict["state_name"] = state_name
        if sac_code is not UNSET:
            field_dict["sac_code"] = sac_code
        if gst_rate is not UNSET:
            field_dict["gst_rate"] = gst_rate
        if pan is not UNSET:
            field_dict["pan"] = pan
        if legal_name is not UNSET:
            field_dict["legal_name"] = legal_name
        if trade_name is not UNSET:
            field_dict["trade_name"] = trade_name
        if address is not UNSET:
            field_dict["address"] = address
        if has_lut is not UNSET:
            field_dict["has_lut"] = has_lut

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        gstin = d.pop("gstin", UNSET)

        state_code = d.pop("state_code", UNSET)

        state_name = d.pop("state_name", UNSET)

        sac_code = d.pop("sac_code", UNSET)

        gst_rate = d.pop("gst_rate", UNSET)

        pan = d.pop("pan", UNSET)

        legal_name = d.pop("legal_name", UNSET)

        trade_name = d.pop("trade_name", UNSET)

        address = d.pop("address", UNSET)

        has_lut = d.pop("has_lut", UNSET)

        gst_config = cls(
            gstin=gstin,
            state_code=state_code,
            state_name=state_name,
            sac_code=sac_code,
            gst_rate=gst_rate,
            pan=pan,
            legal_name=legal_name,
            trade_name=trade_name,
            address=address,
            has_lut=has_lut,
        )

        gst_config.additional_properties = d
        return gst_config

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
