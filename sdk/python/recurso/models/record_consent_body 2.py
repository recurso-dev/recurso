from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.record_consent_body_consent_type import RecordConsentBodyConsentType
from ..types import UNSET, Unset

T = TypeVar("T", bound="RecordConsentBody")


@_attrs_define
class RecordConsentBody:
    """
    Attributes:
        customer_id (UUID):
        consent_type (RecordConsentBodyConsentType):
        granted (bool):
        subscription_id (UUID | Unset):
        consent_text (str | Unset): Exact text shown to the customer. Defaults to the standard recurring-billing text.
    """

    customer_id: UUID
    consent_type: RecordConsentBodyConsentType
    granted: bool
    subscription_id: UUID | Unset = UNSET
    consent_text: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        customer_id = str(self.customer_id)

        consent_type = self.consent_type.value

        granted = self.granted

        subscription_id: str | Unset = UNSET
        if not isinstance(self.subscription_id, Unset):
            subscription_id = str(self.subscription_id)

        consent_text = self.consent_text

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "customer_id": customer_id,
                "consent_type": consent_type,
                "granted": granted,
            }
        )
        if subscription_id is not UNSET:
            field_dict["subscription_id"] = subscription_id
        if consent_text is not UNSET:
            field_dict["consent_text"] = consent_text

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        customer_id = UUID(d.pop("customer_id"))

        consent_type = RecordConsentBodyConsentType(d.pop("consent_type"))

        granted = d.pop("granted")

        _subscription_id = d.pop("subscription_id", UNSET)
        subscription_id: UUID | Unset
        if isinstance(_subscription_id, Unset):
            subscription_id = UNSET
        else:
            subscription_id = UUID(_subscription_id)

        consent_text = d.pop("consent_text", UNSET)

        record_consent_body = cls(
            customer_id=customer_id,
            consent_type=consent_type,
            granted=granted,
            subscription_id=subscription_id,
            consent_text=consent_text,
        )

        record_consent_body.additional_properties = d
        return record_consent_body

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
