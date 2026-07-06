from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.submit_step_result_status import SubmitStepResultStatus
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.cancel_flow_step import CancelFlowStep


T = TypeVar("T", bound="SubmitStepResult")


@_attrs_define
class SubmitStepResult:
    """
    Attributes:
        session_id (UUID | Unset):
        status (SubmitStepResultStatus | Unset):
        next_step (CancelFlowStep | Unset):
        saved_by_offer (bool | Unset):
        completed (bool | Unset):
    """

    session_id: UUID | Unset = UNSET
    status: SubmitStepResultStatus | Unset = UNSET
    next_step: CancelFlowStep | Unset = UNSET
    saved_by_offer: bool | Unset = UNSET
    completed: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        session_id: str | Unset = UNSET
        if not isinstance(self.session_id, Unset):
            session_id = str(self.session_id)

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        next_step: dict[str, Any] | Unset = UNSET
        if not isinstance(self.next_step, Unset):
            next_step = self.next_step.to_dict()

        saved_by_offer = self.saved_by_offer

        completed = self.completed

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if session_id is not UNSET:
            field_dict["session_id"] = session_id
        if status is not UNSET:
            field_dict["status"] = status
        if next_step is not UNSET:
            field_dict["next_step"] = next_step
        if saved_by_offer is not UNSET:
            field_dict["saved_by_offer"] = saved_by_offer
        if completed is not UNSET:
            field_dict["completed"] = completed

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

        _status = d.pop("status", UNSET)
        status: SubmitStepResultStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = SubmitStepResultStatus(_status)

        _next_step = d.pop("next_step", UNSET)
        next_step: CancelFlowStep | Unset
        if isinstance(_next_step, Unset):
            next_step = UNSET
        else:
            next_step = CancelFlowStep.from_dict(_next_step)

        saved_by_offer = d.pop("saved_by_offer", UNSET)

        completed = d.pop("completed", UNSET)

        submit_step_result = cls(
            session_id=session_id,
            status=status,
            next_step=next_step,
            saved_by_offer=saved_by_offer,
            completed=completed,
        )

        submit_step_result.additional_properties = d
        return submit_step_result

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
