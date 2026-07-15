from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.get_tax_nexus_status_response_200_data_states_item import GetTaxNexusStatusResponse200DataStatesItem


T = TypeVar("T", bound="GetTaxNexusStatusResponse200Data")


@_attrs_define
class GetTaxNexusStatusResponse200Data:
    """
    Attributes:
        year (int | Unset):
        dataset_certified (bool | Unset):
        states (list[GetTaxNexusStatusResponse200DataStatesItem] | Unset):
    """

    year: int | Unset = UNSET
    dataset_certified: bool | Unset = UNSET
    states: list[GetTaxNexusStatusResponse200DataStatesItem] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        year = self.year

        dataset_certified = self.dataset_certified

        states: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.states, Unset):
            states = []
            for states_item_data in self.states:
                states_item = states_item_data.to_dict()
                states.append(states_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if year is not UNSET:
            field_dict["year"] = year
        if dataset_certified is not UNSET:
            field_dict["dataset_certified"] = dataset_certified
        if states is not UNSET:
            field_dict["states"] = states

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.get_tax_nexus_status_response_200_data_states_item import (
            GetTaxNexusStatusResponse200DataStatesItem,
        )

        d = dict(src_dict)
        year = d.pop("year", UNSET)

        dataset_certified = d.pop("dataset_certified", UNSET)

        _states = d.pop("states", UNSET)
        states: list[GetTaxNexusStatusResponse200DataStatesItem] | Unset = UNSET
        if _states is not UNSET:
            states = []
            for states_item_data in _states:
                states_item = GetTaxNexusStatusResponse200DataStatesItem.from_dict(states_item_data)

                states.append(states_item)

        get_tax_nexus_status_response_200_data = cls(
            year=year,
            dataset_certified=dataset_certified,
            states=states,
        )

        get_tax_nexus_status_response_200_data.additional_properties = d
        return get_tax_nexus_status_response_200_data

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
