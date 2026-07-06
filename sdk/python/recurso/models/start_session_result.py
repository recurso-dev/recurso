from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.cancel_flow_step import CancelFlowStep


T = TypeVar("T", bound="StartSessionResult")


@_attrs_define
class StartSessionResult:
    """
    Attributes:
        session_id (UUID | Unset):
        flow_id (UUID | Unset):
        steps (list[CancelFlowStep] | Unset):
        first_step (CancelFlowStep | None | Unset):
    """

    session_id: UUID | Unset = UNSET
    flow_id: UUID | Unset = UNSET
    steps: list[CancelFlowStep] | Unset = UNSET
    first_step: CancelFlowStep | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.cancel_flow_step import CancelFlowStep

        session_id: str | Unset = UNSET
        if not isinstance(self.session_id, Unset):
            session_id = str(self.session_id)

        flow_id: str | Unset = UNSET
        if not isinstance(self.flow_id, Unset):
            flow_id = str(self.flow_id)

        steps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.steps, Unset):
            steps = []
            for steps_item_data in self.steps:
                steps_item = steps_item_data.to_dict()
                steps.append(steps_item)

        first_step: dict[str, Any] | None | Unset
        if isinstance(self.first_step, Unset):
            first_step = UNSET
        elif isinstance(self.first_step, CancelFlowStep):
            first_step = self.first_step.to_dict()
        else:
            first_step = self.first_step

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if session_id is not UNSET:
            field_dict["session_id"] = session_id
        if flow_id is not UNSET:
            field_dict["flow_id"] = flow_id
        if steps is not UNSET:
            field_dict["steps"] = steps
        if first_step is not UNSET:
            field_dict["first_step"] = first_step

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.cancel_flow_step import CancelFlowStep

        d = dict(src_dict)
        _session_id = d.pop("session_id", UNSET)
        session_id: UUID | Unset
        if isinstance(_session_id, Unset):
            session_id = UNSET
        else:
            session_id = UUID(_session_id)

        _flow_id = d.pop("flow_id", UNSET)
        flow_id: UUID | Unset
        if isinstance(_flow_id, Unset):
            flow_id = UNSET
        else:
            flow_id = UUID(_flow_id)

        _steps = d.pop("steps", UNSET)
        steps: list[CancelFlowStep] | Unset = UNSET
        if _steps is not UNSET:
            steps = []
            for steps_item_data in _steps:
                steps_item = CancelFlowStep.from_dict(steps_item_data)

                steps.append(steps_item)

        def _parse_first_step(data: object) -> CancelFlowStep | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                first_step_type_0 = CancelFlowStep.from_dict(data)

                return first_step_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(CancelFlowStep | None | Unset, data)

        first_step = _parse_first_step(d.pop("first_step", UNSET))

        start_session_result = cls(
            session_id=session_id,
            flow_id=flow_id,
            steps=steps,
            first_step=first_step,
        )

        start_session_result.additional_properties = d
        return start_session_result

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
