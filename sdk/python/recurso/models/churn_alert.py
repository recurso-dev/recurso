from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ChurnAlert")


@_attrs_define
class ChurnAlert:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        previous_score (int | Unset):
        new_score (int | Unset):
        threshold (int | Unset):
        alert_type (str | Unset):
        acknowledged (bool | Unset):
        created_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    previous_score: int | Unset = UNSET
    new_score: int | Unset = UNSET
    threshold: int | Unset = UNSET
    alert_type: str | Unset = UNSET
    acknowledged: bool | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
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

        previous_score = self.previous_score

        new_score = self.new_score

        threshold = self.threshold

        alert_type = self.alert_type

        acknowledged = self.acknowledged

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
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if previous_score is not UNSET:
            field_dict["previous_score"] = previous_score
        if new_score is not UNSET:
            field_dict["new_score"] = new_score
        if threshold is not UNSET:
            field_dict["threshold"] = threshold
        if alert_type is not UNSET:
            field_dict["alert_type"] = alert_type
        if acknowledged is not UNSET:
            field_dict["acknowledged"] = acknowledged
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

        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        previous_score = d.pop("previous_score", UNSET)

        new_score = d.pop("new_score", UNSET)

        threshold = d.pop("threshold", UNSET)

        alert_type = d.pop("alert_type", UNSET)

        acknowledged = d.pop("acknowledged", UNSET)

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        churn_alert = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            previous_score=previous_score,
            new_score=new_score,
            threshold=threshold,
            alert_type=alert_type,
            acknowledged=acknowledged,
            created_at=created_at,
        )

        churn_alert.additional_properties = d
        return churn_alert

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
