from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.health_response_status import HealthResponseStatus
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.health_response_components import HealthResponseComponents


T = TypeVar("T", bound="HealthResponse")


@_attrs_define
class HealthResponse:
    """
    Attributes:
        status (HealthResponseStatus | Unset):
        version (str | Unset):
        components (HealthResponseComponents | Unset):
    """

    status: HealthResponseStatus | Unset = UNSET
    version: str | Unset = UNSET
    components: HealthResponseComponents | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        version = self.version

        components: dict[str, Any] | Unset = UNSET
        if not isinstance(self.components, Unset):
            components = self.components.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if status is not UNSET:
            field_dict["status"] = status
        if version is not UNSET:
            field_dict["version"] = version
        if components is not UNSET:
            field_dict["components"] = components

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.health_response_components import HealthResponseComponents

        d = dict(src_dict)
        _status = d.pop("status", UNSET)
        status: HealthResponseStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = HealthResponseStatus(_status)

        version = d.pop("version", UNSET)

        _components = d.pop("components", UNSET)
        components: HealthResponseComponents | Unset
        if isinstance(_components, Unset):
            components = UNSET
        else:
            components = HealthResponseComponents.from_dict(_components)

        health_response = cls(
            status=status,
            version=version,
            components=components,
        )

        health_response.additional_properties = d
        return health_response

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
