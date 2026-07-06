from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.cancel_flow_session_status import CancelFlowSessionStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="CancelFlowSession")


@_attrs_define
class CancelFlowSession:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        subscription_id (UUID | Unset):
        flow_id (UUID | Unset):
        status (CancelFlowSessionStatus | Unset):
        current_step_index (int | Unset):
        cancellation_reason (str | Unset):
        feedback (str | Unset):
        offer_presented (Any | Unset): The retention offer shown, if any (arbitrary JSON).
        offer_accepted (bool | Unset):
        saved_by_offer (bool | Unset):
        started_at (datetime.datetime | Unset):
        completed_at (datetime.datetime | None | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    subscription_id: UUID | Unset = UNSET
    flow_id: UUID | Unset = UNSET
    status: CancelFlowSessionStatus | Unset = UNSET
    current_step_index: int | Unset = UNSET
    cancellation_reason: str | Unset = UNSET
    feedback: str | Unset = UNSET
    offer_presented: Any | Unset = UNSET
    offer_accepted: bool | Unset = UNSET
    saved_by_offer: bool | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    completed_at: datetime.datetime | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        subscription_id: str | Unset = UNSET
        if not isinstance(self.subscription_id, Unset):
            subscription_id = str(self.subscription_id)

        flow_id: str | Unset = UNSET
        if not isinstance(self.flow_id, Unset):
            flow_id = str(self.flow_id)

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        current_step_index = self.current_step_index

        cancellation_reason = self.cancellation_reason

        feedback = self.feedback

        offer_presented = self.offer_presented

        offer_accepted = self.offer_accepted

        saved_by_offer = self.saved_by_offer

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        completed_at: None | str | Unset
        if isinstance(self.completed_at, Unset):
            completed_at = UNSET
        elif isinstance(self.completed_at, datetime.datetime):
            completed_at = self.completed_at.isoformat()
        else:
            completed_at = self.completed_at

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if subscription_id is not UNSET:
            field_dict["subscription_id"] = subscription_id
        if flow_id is not UNSET:
            field_dict["flow_id"] = flow_id
        if status is not UNSET:
            field_dict["status"] = status
        if current_step_index is not UNSET:
            field_dict["current_step_index"] = current_step_index
        if cancellation_reason is not UNSET:
            field_dict["cancellation_reason"] = cancellation_reason
        if feedback is not UNSET:
            field_dict["feedback"] = feedback
        if offer_presented is not UNSET:
            field_dict["offer_presented"] = offer_presented
        if offer_accepted is not UNSET:
            field_dict["offer_accepted"] = offer_accepted
        if saved_by_offer is not UNSET:
            field_dict["saved_by_offer"] = saved_by_offer
        if started_at is not UNSET:
            field_dict["started_at"] = started_at
        if completed_at is not UNSET:
            field_dict["completed_at"] = completed_at

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

        _tenant_id = d.pop("tenant_id", UNSET)
        tenant_id: UUID | Unset
        if isinstance(_tenant_id, Unset):
            tenant_id = UNSET
        else:
            tenant_id = UUID(_tenant_id)

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        _subscription_id = d.pop("subscription_id", UNSET)
        subscription_id: UUID | Unset
        if isinstance(_subscription_id, Unset):
            subscription_id = UNSET
        else:
            subscription_id = UUID(_subscription_id)

        _flow_id = d.pop("flow_id", UNSET)
        flow_id: UUID | Unset
        if isinstance(_flow_id, Unset):
            flow_id = UNSET
        else:
            flow_id = UUID(_flow_id)

        _status = d.pop("status", UNSET)
        status: CancelFlowSessionStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = CancelFlowSessionStatus(_status)

        current_step_index = d.pop("current_step_index", UNSET)

        cancellation_reason = d.pop("cancellation_reason", UNSET)

        feedback = d.pop("feedback", UNSET)

        offer_presented = d.pop("offer_presented", UNSET)

        offer_accepted = d.pop("offer_accepted", UNSET)

        saved_by_offer = d.pop("saved_by_offer", UNSET)

        _started_at = d.pop("started_at", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at, Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)

        def _parse_completed_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                completed_at_type_0 = datetime.datetime.fromisoformat(data)

                return completed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        completed_at = _parse_completed_at(d.pop("completed_at", UNSET))

        cancel_flow_session = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            subscription_id=subscription_id,
            flow_id=flow_id,
            status=status,
            current_step_index=current_step_index,
            cancellation_reason=cancellation_reason,
            feedback=feedback,
            offer_presented=offer_presented,
            offer_accepted=offer_accepted,
            saved_by_offer=saved_by_offer,
            started_at=started_at,
            completed_at=completed_at,
        )

        cancel_flow_session.additional_properties = d
        return cancel_flow_session

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
