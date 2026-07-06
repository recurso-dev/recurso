from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="SubmitCancelFlowStepBody")


@_attrs_define
class SubmitCancelFlowStepBody:
    """
    Attributes:
        response (Any): Step-type-specific response payload (e.g. selected survey reason, offer accepted flag).
        step_index (int | Unset):
    """

    response: Any
    step_index: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        response = self.response

        step_index = self.step_index

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "response": response,
            }
        )
        if step_index is not UNSET:
            field_dict["step_index"] = step_index

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        response = d.pop("response")

        step_index = d.pop("step_index", UNSET)

        submit_cancel_flow_step_body = cls(
            response=response,
            step_index=step_index,
        )

        submit_cancel_flow_step_body.additional_properties = d
        return submit_cancel_flow_step_body

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
