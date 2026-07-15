from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.get_tax_nexus_status_response_200_data_states_item_threshold_combinator import (
    GetTaxNexusStatusResponse200DataStatesItemThresholdCombinator,
)
from ..types import UNSET, Unset

T = TypeVar("T", bound="GetTaxNexusStatusResponse200DataStatesItemThreshold")


@_attrs_define
class GetTaxNexusStatusResponse200DataStatesItemThreshold:
    """
    Attributes:
        state_code (str | Unset):
        sales_threshold (int | Unset):
        txn_threshold (int | Unset):
        combinator (GetTaxNexusStatusResponse200DataStatesItemThresholdCombinator | Unset):
        measurement_period (str | Unset):
        certified (bool | Unset):
    """

    state_code: str | Unset = UNSET
    sales_threshold: int | Unset = UNSET
    txn_threshold: int | Unset = UNSET
    combinator: GetTaxNexusStatusResponse200DataStatesItemThresholdCombinator | Unset = UNSET
    measurement_period: str | Unset = UNSET
    certified: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        state_code = self.state_code

        sales_threshold = self.sales_threshold

        txn_threshold = self.txn_threshold

        combinator: str | Unset = UNSET
        if not isinstance(self.combinator, Unset):
            combinator = self.combinator.value

        measurement_period = self.measurement_period

        certified = self.certified

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if state_code is not UNSET:
            field_dict["state_code"] = state_code
        if sales_threshold is not UNSET:
            field_dict["sales_threshold"] = sales_threshold
        if txn_threshold is not UNSET:
            field_dict["txn_threshold"] = txn_threshold
        if combinator is not UNSET:
            field_dict["combinator"] = combinator
        if measurement_period is not UNSET:
            field_dict["measurement_period"] = measurement_period
        if certified is not UNSET:
            field_dict["certified"] = certified

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state_code = d.pop("state_code", UNSET)

        sales_threshold = d.pop("sales_threshold", UNSET)

        txn_threshold = d.pop("txn_threshold", UNSET)

        _combinator = d.pop("combinator", UNSET)
        combinator: GetTaxNexusStatusResponse200DataStatesItemThresholdCombinator | Unset
        if isinstance(_combinator, Unset):
            combinator = UNSET
        else:
            combinator = GetTaxNexusStatusResponse200DataStatesItemThresholdCombinator(_combinator)

        measurement_period = d.pop("measurement_period", UNSET)

        certified = d.pop("certified", UNSET)

        get_tax_nexus_status_response_200_data_states_item_threshold = cls(
            state_code=state_code,
            sales_threshold=sales_threshold,
            txn_threshold=txn_threshold,
            combinator=combinator,
            measurement_period=measurement_period,
            certified=certified,
        )

        get_tax_nexus_status_response_200_data_states_item_threshold.additional_properties = d
        return get_tax_nexus_status_response_200_data_states_item_threshold

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
