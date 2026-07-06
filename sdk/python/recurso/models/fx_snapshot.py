from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.fx_snapshot_source import FXSnapshotSource
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.fx_snapshot_rates import FXSnapshotRates


T = TypeVar("T", bound="FXSnapshot")


@_attrs_define
class FXSnapshot:
    """
    Attributes:
        rates (FXSnapshotRates | Unset):
        source (FXSnapshotSource | Unset):
        as_of (datetime.datetime | Unset):
    """

    rates: FXSnapshotRates | Unset = UNSET
    source: FXSnapshotSource | Unset = UNSET
    as_of: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        rates: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rates, Unset):
            rates = self.rates.to_dict()

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value

        as_of: str | Unset = UNSET
        if not isinstance(self.as_of, Unset):
            as_of = self.as_of.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if rates is not UNSET:
            field_dict["rates"] = rates
        if source is not UNSET:
            field_dict["source"] = source
        if as_of is not UNSET:
            field_dict["as_of"] = as_of

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.fx_snapshot_rates import FXSnapshotRates

        d = dict(src_dict)
        _rates = d.pop("rates", UNSET)
        rates: FXSnapshotRates | Unset
        if isinstance(_rates, Unset):
            rates = UNSET
        else:
            rates = FXSnapshotRates.from_dict(_rates)

        _source = d.pop("source", UNSET)
        source: FXSnapshotSource | Unset
        if isinstance(_source, Unset):
            source = UNSET
        else:
            source = FXSnapshotSource(_source)

        _as_of = d.pop("as_of", UNSET)
        as_of: datetime.datetime | Unset
        if isinstance(_as_of, Unset):
            as_of = UNSET
        else:
            as_of = datetime.datetime.fromisoformat(_as_of)

        fx_snapshot = cls(
            rates=rates,
            source=source,
            as_of=as_of,
        )

        fx_snapshot.additional_properties = d
        return fx_snapshot

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
