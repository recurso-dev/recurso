from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.flow_stats_reason_breakdown import FlowStatsReasonBreakdown


T = TypeVar("T", bound="FlowStats")


@_attrs_define
class FlowStats:
    """
    Attributes:
        total_sessions (int | Unset):
        completed_count (int | Unset):
        saved_count (int | Unset):
        save_rate (float | Unset):
        reason_breakdown (FlowStatsReasonBreakdown | Unset):
        offer_accept_rate (float | Unset):
    """

    total_sessions: int | Unset = UNSET
    completed_count: int | Unset = UNSET
    saved_count: int | Unset = UNSET
    save_rate: float | Unset = UNSET
    reason_breakdown: FlowStatsReasonBreakdown | Unset = UNSET
    offer_accept_rate: float | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        total_sessions = self.total_sessions

        completed_count = self.completed_count

        saved_count = self.saved_count

        save_rate = self.save_rate

        reason_breakdown: dict[str, Any] | Unset = UNSET
        if not isinstance(self.reason_breakdown, Unset):
            reason_breakdown = self.reason_breakdown.to_dict()

        offer_accept_rate = self.offer_accept_rate

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if total_sessions is not UNSET:
            field_dict["total_sessions"] = total_sessions
        if completed_count is not UNSET:
            field_dict["completed_count"] = completed_count
        if saved_count is not UNSET:
            field_dict["saved_count"] = saved_count
        if save_rate is not UNSET:
            field_dict["save_rate"] = save_rate
        if reason_breakdown is not UNSET:
            field_dict["reason_breakdown"] = reason_breakdown
        if offer_accept_rate is not UNSET:
            field_dict["offer_accept_rate"] = offer_accept_rate

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.flow_stats_reason_breakdown import FlowStatsReasonBreakdown

        d = dict(src_dict)
        total_sessions = d.pop("total_sessions", UNSET)

        completed_count = d.pop("completed_count", UNSET)

        saved_count = d.pop("saved_count", UNSET)

        save_rate = d.pop("save_rate", UNSET)

        _reason_breakdown = d.pop("reason_breakdown", UNSET)
        reason_breakdown: FlowStatsReasonBreakdown | Unset
        if isinstance(_reason_breakdown, Unset):
            reason_breakdown = UNSET
        else:
            reason_breakdown = FlowStatsReasonBreakdown.from_dict(_reason_breakdown)

        offer_accept_rate = d.pop("offer_accept_rate", UNSET)

        flow_stats = cls(
            total_sessions=total_sessions,
            completed_count=completed_count,
            saved_count=saved_count,
            save_rate=save_rate,
            reason_breakdown=reason_breakdown,
            offer_accept_rate=offer_accept_rate,
        )

        flow_stats.additional_properties = d
        return flow_stats

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
