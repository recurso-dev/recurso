from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.query_usage_response_200_granularity import QueryUsageResponse200Granularity
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.usage_bucket import UsageBucket


T = TypeVar("T", bound="QueryUsageResponse200")


@_attrs_define
class QueryUsageResponse200:
    """
    Attributes:
        data (list[UsageBucket] | Unset):
        from_ (datetime.datetime | Unset):
        to (datetime.datetime | Unset):
        granularity (QueryUsageResponse200Granularity | Unset):
    """

    data: list[UsageBucket] | Unset = UNSET
    from_: datetime.datetime | Unset = UNSET
    to: datetime.datetime | Unset = UNSET
    granularity: QueryUsageResponse200Granularity | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        data: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = []
            for data_item_data in self.data:
                data_item = data_item_data.to_dict()
                data.append(data_item)

        from_: str | Unset = UNSET
        if not isinstance(self.from_, Unset):
            from_ = self.from_.isoformat()

        to: str | Unset = UNSET
        if not isinstance(self.to, Unset):
            to = self.to.isoformat()

        granularity: str | Unset = UNSET
        if not isinstance(self.granularity, Unset):
            granularity = self.granularity.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if data is not UNSET:
            field_dict["data"] = data
        if from_ is not UNSET:
            field_dict["from"] = from_
        if to is not UNSET:
            field_dict["to"] = to
        if granularity is not UNSET:
            field_dict["granularity"] = granularity

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.usage_bucket import UsageBucket

        d = dict(src_dict)
        _data = d.pop("data", UNSET)
        data: list[UsageBucket] | Unset = UNSET
        if _data is not UNSET:
            data = []
            for data_item_data in _data:
                data_item = UsageBucket.from_dict(data_item_data)

                data.append(data_item)

        _from_ = d.pop("from", UNSET)
        from_: datetime.datetime | Unset
        if isinstance(_from_, Unset):
            from_ = UNSET
        else:
            from_ = datetime.datetime.fromisoformat(_from_)

        _to = d.pop("to", UNSET)
        to: datetime.datetime | Unset
        if isinstance(_to, Unset):
            to = UNSET
        else:
            to = datetime.datetime.fromisoformat(_to)

        _granularity = d.pop("granularity", UNSET)
        granularity: QueryUsageResponse200Granularity | Unset
        if isinstance(_granularity, Unset):
            granularity = UNSET
        else:
            granularity = QueryUsageResponse200Granularity(_granularity)

        query_usage_response_200 = cls(
            data=data,
            from_=from_,
            to=to,
            granularity=granularity,
        )

        query_usage_response_200.additional_properties = d
        return query_usage_response_200

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
