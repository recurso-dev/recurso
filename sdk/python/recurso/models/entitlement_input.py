from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.entitlement_input_kind import EntitlementInputKind
from ..types import UNSET, Unset

T = TypeVar("T", bound="EntitlementInput")


@_attrs_define
class EntitlementInput:
    """
    Attributes:
        feature_key (str):
        kind (EntitlementInputKind):
        bool_value (bool | Unset):
        limit_value (int | Unset):
    """

    feature_key: str
    kind: EntitlementInputKind
    bool_value: bool | Unset = UNSET
    limit_value: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        feature_key = self.feature_key

        kind = self.kind.value

        bool_value = self.bool_value

        limit_value = self.limit_value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "feature_key": feature_key,
                "kind": kind,
            }
        )
        if bool_value is not UNSET:
            field_dict["bool_value"] = bool_value
        if limit_value is not UNSET:
            field_dict["limit_value"] = limit_value

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        feature_key = d.pop("feature_key")

        kind = EntitlementInputKind(d.pop("kind"))

        bool_value = d.pop("bool_value", UNSET)

        limit_value = d.pop("limit_value", UNSET)

        entitlement_input = cls(
            feature_key=feature_key,
            kind=kind,
            bool_value=bool_value,
            limit_value=limit_value,
        )

        entitlement_input.additional_properties = d
        return entitlement_input

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
