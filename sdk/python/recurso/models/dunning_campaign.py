from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.dunning_campaign_step import DunningCampaignStep


T = TypeVar("T", bound="DunningCampaign")


@_attrs_define
class DunningCampaign:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        name (str | Unset):
        is_active (bool | Unset):
        trigger_event (str | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
        steps (list[DunningCampaignStep] | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    name: str | Unset = UNSET
    is_active: bool | Unset = UNSET
    trigger_event: str | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    steps: list[DunningCampaignStep] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        name = self.name

        is_active = self.is_active

        trigger_event = self.trigger_event

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        steps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.steps, Unset):
            steps = []
            for steps_item_data in self.steps:
                steps_item = steps_item_data.to_dict()
                steps.append(steps_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if name is not UNSET:
            field_dict["name"] = name
        if is_active is not UNSET:
            field_dict["is_active"] = is_active
        if trigger_event is not UNSET:
            field_dict["trigger_event"] = trigger_event
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at
        if steps is not UNSET:
            field_dict["steps"] = steps

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.dunning_campaign_step import DunningCampaignStep

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

        name = d.pop("name", UNSET)

        is_active = d.pop("is_active", UNSET)

        trigger_event = d.pop("trigger_event", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        _updated_at = d.pop("updated_at", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at, Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)

        _steps = d.pop("steps", UNSET)
        steps: list[DunningCampaignStep] | Unset = UNSET
        if _steps is not UNSET:
            steps = []
            for steps_item_data in _steps:
                steps_item = DunningCampaignStep.from_dict(steps_item_data)

                steps.append(steps_item)

        dunning_campaign = cls(
            id=id,
            tenant_id=tenant_id,
            name=name,
            is_active=is_active,
            trigger_event=trigger_event,
            created_at=created_at,
            updated_at=updated_at,
            steps=steps,
        )

        dunning_campaign.additional_properties = d
        return dunning_campaign

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
