from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.update_dunning_campaign_step_body_channel import UpdateDunningCampaignStepBodyChannel
from ..types import UNSET, Unset

T = TypeVar("T", bound="UpdateDunningCampaignStepBody")


@_attrs_define
class UpdateDunningCampaignStepBody:
    """
    Attributes:
        step_order (int | Unset):
        channel (UpdateDunningCampaignStepBodyChannel | Unset):
        delay_hours (int | Unset):
        template_name (str | Unset):
        subject (str | Unset):
        body (str | Unset):
        is_payment_wall (bool | Unset):
    """

    step_order: int | Unset = UNSET
    channel: UpdateDunningCampaignStepBodyChannel | Unset = UNSET
    delay_hours: int | Unset = UNSET
    template_name: str | Unset = UNSET
    subject: str | Unset = UNSET
    body: str | Unset = UNSET
    is_payment_wall: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        step_order = self.step_order

        channel: str | Unset = UNSET
        if not isinstance(self.channel, Unset):
            channel = self.channel.value

        delay_hours = self.delay_hours

        template_name = self.template_name

        subject = self.subject

        body = self.body

        is_payment_wall = self.is_payment_wall

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
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

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        step_order = d.pop("step_order", UNSET)

        _channel = d.pop("channel", UNSET)
        channel: UpdateDunningCampaignStepBodyChannel | Unset
        if isinstance(_channel, Unset):
            channel = UNSET
        else:
            channel = UpdateDunningCampaignStepBodyChannel(_channel)

        delay_hours = d.pop("delay_hours", UNSET)

        template_name = d.pop("template_name", UNSET)

        subject = d.pop("subject", UNSET)

        body = d.pop("body", UNSET)

        is_payment_wall = d.pop("is_payment_wall", UNSET)

        update_dunning_campaign_step_body = cls(
            step_order=step_order,
            channel=channel,
            delay_hours=delay_hours,
            template_name=template_name,
            subject=subject,
            body=body,
            is_payment_wall=is_payment_wall,
        )

        update_dunning_campaign_step_body.additional_properties = d
        return update_dunning_campaign_step_body

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
