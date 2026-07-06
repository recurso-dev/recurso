from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.dunning_campaign_step_channel import DunningCampaignStepChannel
from ..types import UNSET, Unset

T = TypeVar("T", bound="DunningCampaignStep")


@_attrs_define
class DunningCampaignStep:
    """
    Attributes:
        id (UUID | Unset):
        campaign_id (UUID | Unset):
        step_order (int | Unset):
        channel (DunningCampaignStepChannel | Unset):
        delay_hours (int | Unset):
        template_name (str | Unset):
        subject (str | Unset):
        body (str | Unset):
        is_payment_wall (bool | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    campaign_id: UUID | Unset = UNSET
    step_order: int | Unset = UNSET
    channel: DunningCampaignStepChannel | Unset = UNSET
    delay_hours: int | Unset = UNSET
    template_name: str | Unset = UNSET
    subject: str | Unset = UNSET
    body: str | Unset = UNSET
    is_payment_wall: bool | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        campaign_id: str | Unset = UNSET
        if not isinstance(self.campaign_id, Unset):
            campaign_id = str(self.campaign_id)

        step_order = self.step_order

        channel: str | Unset = UNSET
        if not isinstance(self.channel, Unset):
            channel = self.channel.value

        delay_hours = self.delay_hours

        template_name = self.template_name

        subject = self.subject

        body = self.body

        is_payment_wall = self.is_payment_wall

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if campaign_id is not UNSET:
            field_dict["campaign_id"] = campaign_id
        if step_order is not UNSET:
            field_dict["step_order"] = step_order
        if channel is not UNSET:
            field_dict["channel"] = channel
        if delay_hours is not UNSET:
            field_dict["delay_hours"] = delay_hours
        if template_name is not UNSET:
            field_dict["template_name"] = template_name
        if subject is not UNSET:
            field_dict["subject"] = subject
        if body is not UNSET:
            field_dict["body"] = body
        if is_payment_wall is not UNSET:
            field_dict["is_payment_wall"] = is_payment_wall
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

        _campaign_id = d.pop("campaign_id", UNSET)
        campaign_id: UUID | Unset
        if isinstance(_campaign_id, Unset):
            campaign_id = UNSET
        else:
            campaign_id = UUID(_campaign_id)

        step_order = d.pop("step_order", UNSET)

        _channel = d.pop("channel", UNSET)
        channel: DunningCampaignStepChannel | Unset
        if isinstance(_channel, Unset):
            channel = UNSET
        else:
            channel = DunningCampaignStepChannel(_channel)

        delay_hours = d.pop("delay_hours", UNSET)

        template_name = d.pop("template_name", UNSET)

        subject = d.pop("subject", UNSET)

        body = d.pop("body", UNSET)

        is_payment_wall = d.pop("is_payment_wall", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        dunning_campaign_step = cls(
            id=id,
            campaign_id=campaign_id,
            step_order=step_order,
            channel=channel,
            delay_hours=delay_hours,
            template_name=template_name,
            subject=subject,
            body=body,
            is_payment_wall=is_payment_wall,
            created_at=created_at,
        )

        dunning_campaign_step.additional_properties = d
        return dunning_campaign_step

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
