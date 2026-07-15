from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.get_tax_nexus_status_response_200_data_states_item_nexus_type import (
    GetTaxNexusStatusResponse200DataStatesItemNexusType,
)
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.get_tax_nexus_status_response_200_data_states_item_threshold import (
        GetTaxNexusStatusResponse200DataStatesItemThreshold,
    )


T = TypeVar("T", bound="GetTaxNexusStatusResponse200DataStatesItem")


@_attrs_define
class GetTaxNexusStatusResponse200DataStatesItem:
    """
    Attributes:
        state_code (str | Unset):
        nexus_type (GetTaxNexusStatusResponse200DataStatesItemNexusType | Unset):
        established_at (datetime.datetime | Unset):
        taxable_sales (int | Unset): USD cents, calendar-year sum of invoice subtotals.
        txn_count (int | Unset):
        threshold (GetTaxNexusStatusResponse200DataStatesItemThreshold | Unset):
        proximity_pct (int | Unset):
        crossed (bool | Unset):
    """

    state_code: str | Unset = UNSET
    nexus_type: GetTaxNexusStatusResponse200DataStatesItemNexusType | Unset = UNSET
    established_at: datetime.datetime | Unset = UNSET
    taxable_sales: int | Unset = UNSET
    txn_count: int | Unset = UNSET
    threshold: GetTaxNexusStatusResponse200DataStatesItemThreshold | Unset = UNSET
    proximity_pct: int | Unset = UNSET
    crossed: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        state_code = self.state_code

        nexus_type: str | Unset = UNSET
        if not isinstance(self.nexus_type, Unset):
            nexus_type = self.nexus_type.value

        established_at: str | Unset = UNSET
        if not isinstance(self.established_at, Unset):
            established_at = self.established_at.isoformat()

        taxable_sales = self.taxable_sales

        txn_count = self.txn_count

        threshold: dict[str, Any] | Unset = UNSET
        if not isinstance(self.threshold, Unset):
            threshold = self.threshold.to_dict()

        proximity_pct = self.proximity_pct

        crossed = self.crossed

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if state_code is not UNSET:
            field_dict["state_code"] = state_code
        if nexus_type is not UNSET:
            field_dict["nexus_type"] = nexus_type
        if established_at is not UNSET:
            field_dict["established_at"] = established_at
        if taxable_sales is not UNSET:
            field_dict["taxable_sales"] = taxable_sales
        if txn_count is not UNSET:
            field_dict["txn_count"] = txn_count
        if threshold is not UNSET:
            field_dict["threshold"] = threshold
        if proximity_pct is not UNSET:
            field_dict["proximity_pct"] = proximity_pct
        if crossed is not UNSET:
            field_dict["crossed"] = crossed

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.get_tax_nexus_status_response_200_data_states_item_threshold import (
            GetTaxNexusStatusResponse200DataStatesItemThreshold,
        )

        d = dict(src_dict)
        state_code = d.pop("state_code", UNSET)

        _nexus_type = d.pop("nexus_type", UNSET)
        nexus_type: GetTaxNexusStatusResponse200DataStatesItemNexusType | Unset
        if isinstance(_nexus_type, Unset):
            nexus_type = UNSET
        else:
            nexus_type = GetTaxNexusStatusResponse200DataStatesItemNexusType(_nexus_type)

        _established_at = d.pop("established_at", UNSET)
        established_at: datetime.datetime | Unset
        if isinstance(_established_at, Unset):
            established_at = UNSET
        else:
            established_at = datetime.datetime.fromisoformat(_established_at)

        taxable_sales = d.pop("taxable_sales", UNSET)

        txn_count = d.pop("txn_count", UNSET)

        _threshold = d.pop("threshold", UNSET)
        threshold: GetTaxNexusStatusResponse200DataStatesItemThreshold | Unset
        if isinstance(_threshold, Unset):
            threshold = UNSET
        else:
            threshold = GetTaxNexusStatusResponse200DataStatesItemThreshold.from_dict(_threshold)

        proximity_pct = d.pop("proximity_pct", UNSET)

        crossed = d.pop("crossed", UNSET)

        get_tax_nexus_status_response_200_data_states_item = cls(
            state_code=state_code,
            nexus_type=nexus_type,
            established_at=established_at,
            taxable_sales=taxable_sales,
            txn_count=txn_count,
            threshold=threshold,
            proximity_pct=proximity_pct,
            crossed=crossed,
        )

        get_tax_nexus_status_response_200_data_states_item.additional_properties = d
        return get_tax_nexus_status_response_200_data_states_item

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
