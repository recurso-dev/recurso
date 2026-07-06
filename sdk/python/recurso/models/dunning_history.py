from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.dunning_history_outcome import DunningHistoryOutcome
from ..types import UNSET, Unset

T = TypeVar("T", bound="DunningHistory")


@_attrs_define
class DunningHistory:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        invoice_id (UUID | Unset):
        context_key (str | Unset):
        action_id (str | Unset):
        retry_interval (int | Unset): Retry interval in seconds.
        outcome (DunningHistoryOutcome | Unset):
        reward (float | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    invoice_id: UUID | Unset = UNSET
    context_key: str | Unset = UNSET
    action_id: str | Unset = UNSET
    retry_interval: int | Unset = UNSET
    outcome: DunningHistoryOutcome | Unset = UNSET
    reward: float | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        context_key = self.context_key

        action_id = self.action_id

        retry_interval = self.retry_interval

        outcome: str | Unset = UNSET
        if not isinstance(self.outcome, Unset):
            outcome = self.outcome.value

        reward = self.reward

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if context_key is not UNSET:
            field_dict["context_key"] = context_key
        if action_id is not UNSET:
            field_dict["action_id"] = action_id
        if retry_interval is not UNSET:
            field_dict["retry_interval"] = retry_interval
        if outcome is not UNSET:
            field_dict["outcome"] = outcome
        if reward is not UNSET:
            field_dict["reward"] = reward
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

        _tenant_id = d.pop("tenant_id", UNSET)
        tenant_id: UUID | Unset
        if isinstance(_tenant_id, Unset):
            tenant_id = UNSET
        else:
            tenant_id = UUID(_tenant_id)

        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        context_key = d.pop("context_key", UNSET)

        action_id = d.pop("action_id", UNSET)

        retry_interval = d.pop("retry_interval", UNSET)

        _outcome = d.pop("outcome", UNSET)
        outcome: DunningHistoryOutcome | Unset
        if isinstance(_outcome, Unset):
            outcome = UNSET
        else:
            outcome = DunningHistoryOutcome(_outcome)

        reward = d.pop("reward", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        dunning_history = cls(
            id=id,
            tenant_id=tenant_id,
            invoice_id=invoice_id,
            context_key=context_key,
            action_id=action_id,
            retry_interval=retry_interval,
            outcome=outcome,
            reward=reward,
            created_at=created_at,
        )

        dunning_history.additional_properties = d
        return dunning_history

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
