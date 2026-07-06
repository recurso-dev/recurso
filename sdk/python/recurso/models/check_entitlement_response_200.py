from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CheckEntitlementResponse200")


@_attrs_define
class CheckEntitlementResponse200:
    """
    Attributes:
        feature_key (str | Unset):
        granted (bool | Unset):
        limit_value (int | None | Unset):
    """

    feature_key: str | Unset = UNSET
    granted: bool | Unset = UNSET
    limit_value: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        feature_key = self.feature_key

        granted = self.granted

        limit_value: int | None | Unset
        if isinstance(self.limit_value, Unset):
            limit_value = UNSET
        else:
            limit_value = self.limit_value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if feature_key is not UNSET:
            field_dict["feature_key"] = feature_key
        if granted is not UNSET:
            field_dict["granted"] = granted
        if limit_value is not UNSET:
            field_dict["limit_value"] = limit_value

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        feature_key = d.pop("feature_key", UNSET)

        granted = d.pop("granted", UNSET)

        def _parse_limit_value(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        limit_value = _parse_limit_value(d.pop("limit_value", UNSET))

        check_entitlement_response_200 = cls(
            feature_key=feature_key,
            granted=granted,
            limit_value=limit_value,
        )

        check_entitlement_response_200.additional_properties = d
        return check_entitlement_response_200

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
