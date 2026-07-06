from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.cancel_flow_step_type import CancelFlowStepType
from ..types import UNSET, Unset

T = TypeVar("T", bound="CancelFlowStep")


@_attrs_define
class CancelFlowStep:
    """
    Attributes:
        id (UUID | Unset):
        flow_id (UUID | Unset):
        step_order (int | Unset):
        step_type (CancelFlowStepType | Unset):
        config (Any | Unset): Step-type-specific configuration (arbitrary JSON).
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    flow_id: UUID | Unset = UNSET
    step_order: int | Unset = UNSET
    step_type: CancelFlowStepType | Unset = UNSET
    config: Any | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        flow_id: str | Unset = UNSET
        if not isinstance(self.flow_id, Unset):
            flow_id = str(self.flow_id)

        step_order = self.step_order

        step_type: str | Unset = UNSET
        if not isinstance(self.step_type, Unset):
            step_type = self.step_type.value

        config = self.config

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if flow_id is not UNSET:
            field_dict["flow_id"] = flow_id
        if step_order is not UNSET:
            field_dict["step_order"] = step_order
        if step_type is not UNSET:
            field_dict["step_type"] = step_type
        if config is not UNSET:
            field_dict["config"] = config
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _id = d.pop("id", UNSET)
        id: UUID | Unset
        if isinstance(_id, Unset):
            id = UNSET
        else:
            id = UUID(_id)

        _flow_id = d.pop("flow_id", UNSET)
        flow_id: UUID | Unset
        if isinstance(_flow_id, Unset):
            flow_id = UNSET
        else:
            flow_id = UUID(_flow_id)

        step_order = d.pop("step_order", UNSET)

        _step_type = d.pop("step_type", UNSET)
        step_type: CancelFlowStepType | Unset
        if isinstance(_step_type, Unset):
            step_type = UNSET
        else:
            step_type = CancelFlowStepType(_step_type)

        config = d.pop("config", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        cancel_flow_step = cls(
            id=id,
            flow_id=flow_id,
            step_order=step_order,
            step_type=step_type,
            config=config,
            created_at=created_at,
        )

        cancel_flow_step.additional_properties = d
        return cancel_flow_step

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
