from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.entitlement_input import EntitlementInput


T = TypeVar("T", bound="SetPlanEntitlementsBody")


@_attrs_define
class SetPlanEntitlementsBody:
    """
    Attributes:
        entitlements (list[EntitlementInput] | Unset):
    """

    entitlements: list[EntitlementInput] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        entitlements: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.entitlements, Unset):
            entitlements = []
            for entitlements_item_data in self.entitlements:
                entitlements_item = entitlements_item_data.to_dict()
                entitlements.append(entitlements_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if entitlements is not UNSET:
            field_dict["entitlements"] = entitlements

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.entitlement_input import EntitlementInput

        d = dict(src_dict)
        _entitlements = d.pop("entitlements", UNSET)
        entitlements: list[EntitlementInput] | Unset = UNSET
        if _entitlements is not UNSET:
            entitlements = []
            for entitlements_item_data in _entitlements:
                entitlements_item = EntitlementInput.from_dict(entitlements_item_data)

                entitlements.append(entitlements_item)

        set_plan_entitlements_body = cls(
            entitlements=entitlements,
        )

        set_plan_entitlements_body.additional_properties = d
        return set_plan_entitlements_body

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
