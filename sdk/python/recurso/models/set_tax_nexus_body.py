from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.set_tax_nexus_body_states_item import SetTaxNexusBodyStatesItem


T = TypeVar("T", bound="SetTaxNexusBody")


@_attrs_define
class SetTaxNexusBody:
    """
    Attributes:
        states (list[SetTaxNexusBodyStatesItem] | Unset):
    """

    states: list[SetTaxNexusBodyStatesItem] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        states: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.states, Unset):
            states = []
            for states_item_data in self.states:
                states_item = states_item_data.to_dict()
                states.append(states_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if states is not UNSET:
            field_dict["states"] = states

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.set_tax_nexus_body_states_item import SetTaxNexusBodyStatesItem

        d = dict(src_dict)
        _states = d.pop("states", UNSET)
        states: list[SetTaxNexusBodyStatesItem] | Unset = UNSET
        if _states is not UNSET:
            states = []
            for states_item_data in _states:
                states_item = SetTaxNexusBodyStatesItem.from_dict(states_item_data)

                states.append(states_item)

        set_tax_nexus_body = cls(
            states=states,
        )

        set_tax_nexus_body.additional_properties = d
        return set_tax_nexus_body

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
