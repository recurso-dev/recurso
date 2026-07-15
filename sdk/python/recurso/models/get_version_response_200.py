from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.get_version_response_200_gateway_mode import GetVersionResponse200GatewayMode
from ..types import UNSET, Unset

T = TypeVar("T", bound="GetVersionResponse200")


@_attrs_define
class GetVersionResponse200:
    """
    Attributes:
        version (str | Unset):
        gateway_mode (GetVersionResponse200GatewayMode | Unset): Whether the configured payment gateways use test or
            live keys.
    """

    version: str | Unset = UNSET
    gateway_mode: GetVersionResponse200GatewayMode | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        version = self.version

        gateway_mode: str | Unset = UNSET
        if not isinstance(self.gateway_mode, Unset):
            gateway_mode = self.gateway_mode.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if version is not UNSET:
            field_dict["version"] = version
        if gateway_mode is not UNSET:
            field_dict["gateway_mode"] = gateway_mode

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version", UNSET)

        _gateway_mode = d.pop("gateway_mode", UNSET)
        gateway_mode: GetVersionResponse200GatewayMode | Unset
        if isinstance(_gateway_mode, Unset):
            gateway_mode = UNSET
        else:
            gateway_mode = GetVersionResponse200GatewayMode(_gateway_mode)

        get_version_response_200 = cls(
            version=version,
            gateway_mode=gateway_mode,
        )

        get_version_response_200.additional_properties = d
        return get_version_response_200

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
