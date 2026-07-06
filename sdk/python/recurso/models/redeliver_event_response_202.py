from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.redeliver_event_response_202_data import RedeliverEventResponse202Data


T = TypeVar("T", bound="RedeliverEventResponse202")


@_attrs_define
class RedeliverEventResponse202:
    """
    Attributes:
        data (RedeliverEventResponse202Data | Unset):
    """

    data: RedeliverEventResponse202Data | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        data: dict[str, Any] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = self.data.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if data is not UNSET:
            field_dict["data"] = data

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.redeliver_event_response_202_data import RedeliverEventResponse202Data

        d = dict(src_dict)
        _data = d.pop("data", UNSET)
        data: RedeliverEventResponse202Data | Unset
        if isinstance(_data, Unset):
            data = UNSET
        else:
            data = RedeliverEventResponse202Data.from_dict(_data)

        redeliver_event_response_202 = cls(
            data=data,
        )

        redeliver_event_response_202.additional_properties = d
        return redeliver_event_response_202

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
