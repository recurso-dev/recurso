from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.cancel_flow_step_type import CancelFlowStepType
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.update_cancel_flow_step_body_config import UpdateCancelFlowStepBodyConfig


T = TypeVar("T", bound="UpdateCancelFlowStepBody")


@_attrs_define
class UpdateCancelFlowStepBody:
    """
    Attributes:
        step_order (int | Unset):
        step_type (CancelFlowStepType | Unset):
        config (UpdateCancelFlowStepBodyConfig | Unset):
    """

    step_order: int | Unset = UNSET
    step_type: CancelFlowStepType | Unset = UNSET
    config: UpdateCancelFlowStepBodyConfig | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        step_order = self.step_order

        step_type: str | Unset = UNSET
        if not isinstance(self.step_type, Unset):
            step_type = self.step_type.value

        config: dict[str, Any] | Unset = UNSET
        if not isinstance(self.config, Unset):
            config = self.config.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if step_order is not UNSET:
            field_dict["step_order"] = step_order
        if step_type is not UNSET:
            field_dict["step_type"] = step_type
        if config is not UNSET:
            field_dict["config"] = config

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.update_cancel_flow_step_body_config import UpdateCancelFlowStepBodyConfig

        d = dict(src_dict)
        step_order = d.pop("step_order", UNSET)

        _step_type = d.pop("step_type", UNSET)
        step_type: CancelFlowStepType | Unset
        if isinstance(_step_type, Unset):
            step_type = UNSET
        else:
            step_type = CancelFlowStepType(_step_type)

        _config = d.pop("config", UNSET)
        config: UpdateCancelFlowStepBodyConfig | Unset
        if isinstance(_config, Unset):
            config = UNSET
        else:
            config = UpdateCancelFlowStepBodyConfig.from_dict(_config)

        update_cancel_flow_step_body = cls(
            step_order=step_order,
            step_type=step_type,
            config=config,
        )

        update_cancel_flow_step_body.additional_properties = d
        return update_cancel_flow_step_body

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
