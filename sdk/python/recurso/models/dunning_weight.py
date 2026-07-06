from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="DunningWeight")


@_attrs_define
class DunningWeight:
    """
    Attributes:
        context_key (str | Unset): Customer context bucket the weight was learned for.
        action_id (str | Unset): Retry action identifier (e.g. 1h, 24h, 3d, 7d).
        average_reward (float | Unset): Learned success rate (0.0-1.0).
        sample_count (int | Unset):
        updated_at (datetime.datetime | Unset):
    """

    context_key: str | Unset = UNSET
    action_id: str | Unset = UNSET
    average_reward: float | Unset = UNSET
    sample_count: int | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        context_key = self.context_key

        action_id = self.action_id

        average_reward = self.average_reward

        sample_count = self.sample_count

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if context_key is not UNSET:
            field_dict["context_key"] = context_key
        if action_id is not UNSET:
            field_dict["action_id"] = action_id
        if average_reward is not UNSET:
            field_dict["average_reward"] = average_reward
        if sample_count is not UNSET:
            field_dict["sample_count"] = sample_count
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        context_key = d.pop("context_key", UNSET)

        action_id = d.pop("action_id", UNSET)

        average_reward = d.pop("average_reward", UNSET)

        sample_count = d.pop("sample_count", UNSET)

        _updated_at = d.pop("updated_at", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at, Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)

        dunning_weight = cls(
            context_key=context_key,
            action_id=action_id,
            average_reward=average_reward,
            sample_count=sample_count,
            updated_at=updated_at,
        )

        dunning_weight.additional_properties = d
        return dunning_weight

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
