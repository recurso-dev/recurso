from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.mandate_frequency import MandateFrequency
from ..models.mandate_status import MandateStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="Mandate")


@_attrs_define
class Mandate:
    """
    Attributes:
        id (UUID | Unset):
        tenant_id (UUID | Unset):
        customer_id (UUID | Unset):
        subscription_id (None | Unset | UUID):
        mandate_type (str | Unset):
        payment_method (str | Unset):
        vpa (str | Unset):
        razorpay_token_id (str | Unset):
        razorpay_subscription_id (str | Unset):
        razorpay_customer_id (str | Unset):
        max_amount (int | Unset):
        frequency (MandateFrequency | Unset):
        status (MandateStatus | Unset):
        authorized_at (datetime.datetime | None | Unset):
        activated_at (datetime.datetime | None | Unset):
        revoked_at (datetime.datetime | None | Unset):
        last_debit_at (datetime.datetime | None | Unset):
        next_debit_at (datetime.datetime | None | Unset):
        pre_debit_notified (bool | Unset):
        created_at (datetime.datetime | Unset):
        updated_at (datetime.datetime | Unset):
    """

    id: UUID | Unset = UNSET
    tenant_id: UUID | Unset = UNSET
    customer_id: UUID | Unset = UNSET
    subscription_id: None | Unset | UUID = UNSET
    mandate_type: str | Unset = UNSET
    payment_method: str | Unset = UNSET
    vpa: str | Unset = UNSET
    razorpay_token_id: str | Unset = UNSET
    razorpay_subscription_id: str | Unset = UNSET
    razorpay_customer_id: str | Unset = UNSET
    max_amount: int | Unset = UNSET
    frequency: MandateFrequency | Unset = UNSET
    status: MandateStatus | Unset = UNSET
    authorized_at: datetime.datetime | None | Unset = UNSET
    activated_at: datetime.datetime | None | Unset = UNSET
    revoked_at: datetime.datetime | None | Unset = UNSET
    last_debit_at: datetime.datetime | None | Unset = UNSET
    next_debit_at: datetime.datetime | None | Unset = UNSET
    pre_debit_notified: bool | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
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

        subscription_id: None | str | Unset
        if isinstance(self.subscription_id, Unset):
            subscription_id = UNSET
        elif isinstance(self.subscription_id, UUID):
            subscription_id = str(self.subscription_id)
        else:
            subscription_id = self.subscription_id

        mandate_type = self.mandate_type

        payment_method = self.payment_method

        vpa = self.vpa

        razorpay_token_id = self.razorpay_token_id

        razorpay_subscription_id = self.razorpay_subscription_id

        razorpay_customer_id = self.razorpay_customer_id

        max_amount = self.max_amount

        frequency: str | Unset = UNSET
        if not isinstance(self.frequency, Unset):
            frequency = self.frequency.value

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        authorized_at: None | str | Unset
        if isinstance(self.authorized_at, Unset):
            authorized_at = UNSET
        elif isinstance(self.authorized_at, datetime.datetime):
            authorized_at = self.authorized_at.isoformat()
        else:
            authorized_at = self.authorized_at

        activated_at: None | str | Unset
        if isinstance(self.activated_at, Unset):
            activated_at = UNSET
        elif isinstance(self.activated_at, datetime.datetime):
            activated_at = self.activated_at.isoformat()
        else:
            activated_at = self.activated_at

        revoked_at: None | str | Unset
        if isinstance(self.revoked_at, Unset):
            revoked_at = UNSET
        elif isinstance(self.revoked_at, datetime.datetime):
            revoked_at = self.revoked_at.isoformat()
        else:
            revoked_at = self.revoked_at

        last_debit_at: None | str | Unset
        if isinstance(self.last_debit_at, Unset):
            last_debit_at = UNSET
        elif isinstance(self.last_debit_at, datetime.datetime):
            last_debit_at = self.last_debit_at.isoformat()
        else:
            last_debit_at = self.last_debit_at

        next_debit_at: None | str | Unset
        if isinstance(self.next_debit_at, Unset):
            next_debit_at = UNSET
        elif isinstance(self.next_debit_at, datetime.datetime):
            next_debit_at = self.next_debit_at.isoformat()
        else:
            next_debit_at = self.next_debit_at

        pre_debit_notified = self.pre_debit_notified

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

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
        if mandate_type is not UNSET:
            field_dict["mandate_type"] = mandate_type
        if payment_method is not UNSET:
            field_dict["payment_method"] = payment_method
        if vpa is not UNSET:
            field_dict["vpa"] = vpa
        if razorpay_token_id is not UNSET:
            field_dict["razorpay_token_id"] = razorpay_token_id
        if razorpay_subscription_id is not UNSET:
            field_dict["razorpay_subscription_id"] = razorpay_subscription_id
        if razorpay_customer_id is not UNSET:
            field_dict["razorpay_customer_id"] = razorpay_customer_id
        if max_amount is not UNSET:
            field_dict["max_amount"] = max_amount
        if frequency is not UNSET:
            field_dict["frequency"] = frequency
        if status is not UNSET:
            field_dict["status"] = status
        if authorized_at is not UNSET:
            field_dict["authorized_at"] = authorized_at
        if activated_at is not UNSET:
            field_dict["activated_at"] = activated_at
        if revoked_at is not UNSET:
            field_dict["revoked_at"] = revoked_at
        if last_debit_at is not UNSET:
            field_dict["last_debit_at"] = last_debit_at
        if next_debit_at is not UNSET:
            field_dict["next_debit_at"] = next_debit_at
        if pre_debit_notified is not UNSET:
            field_dict["pre_debit_notified"] = pre_debit_notified
        if created_at is not UNSET:
            field_dict["created_at"] = created_at
        if updated_at is not UNSET:
            field_dict["updated_at"] = updated_at

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

        def _parse_subscription_id(data: object) -> None | Unset | UUID:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                subscription_id_type_0 = UUID(data)

                return subscription_id_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UUID, data)

        subscription_id = _parse_subscription_id(d.pop("subscription_id", UNSET))

        mandate_type = d.pop("mandate_type", UNSET)

        payment_method = d.pop("payment_method", UNSET)

        vpa = d.pop("vpa", UNSET)

        razorpay_token_id = d.pop("razorpay_token_id", UNSET)

        razorpay_subscription_id = d.pop("razorpay_subscription_id", UNSET)

        razorpay_customer_id = d.pop("razorpay_customer_id", UNSET)

        max_amount = d.pop("max_amount", UNSET)

        _frequency = d.pop("frequency", UNSET)
        frequency: MandateFrequency | Unset
        if isinstance(_frequency, Unset):
            frequency = UNSET
        else:
            frequency = MandateFrequency(_frequency)

        _status = d.pop("status", UNSET)
        status: MandateStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = MandateStatus(_status)

        def _parse_authorized_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                authorized_at_type_0 = datetime.datetime.fromisoformat(data)

                return authorized_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        authorized_at = _parse_authorized_at(d.pop("authorized_at", UNSET))

        def _parse_activated_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                activated_at_type_0 = datetime.datetime.fromisoformat(data)

                return activated_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        activated_at = _parse_activated_at(d.pop("activated_at", UNSET))

        def _parse_revoked_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                revoked_at_type_0 = datetime.datetime.fromisoformat(data)

                return revoked_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        revoked_at = _parse_revoked_at(d.pop("revoked_at", UNSET))

        def _parse_last_debit_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_debit_at_type_0 = datetime.datetime.fromisoformat(data)

                return last_debit_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        last_debit_at = _parse_last_debit_at(d.pop("last_debit_at", UNSET))

        def _parse_next_debit_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                next_debit_at_type_0 = datetime.datetime.fromisoformat(data)

                return next_debit_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        next_debit_at = _parse_next_debit_at(d.pop("next_debit_at", UNSET))

        pre_debit_notified = d.pop("pre_debit_notified", UNSET)

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

        mandate = cls(
            id=id,
            tenant_id=tenant_id,
            customer_id=customer_id,
            subscription_id=subscription_id,
            mandate_type=mandate_type,
            payment_method=payment_method,
            vpa=vpa,
            razorpay_token_id=razorpay_token_id,
            razorpay_subscription_id=razorpay_subscription_id,
            razorpay_customer_id=razorpay_customer_id,
            max_amount=max_amount,
            frequency=frequency,
            status=status,
            authorized_at=authorized_at,
            activated_at=activated_at,
            revoked_at=revoked_at,
            last_debit_at=last_debit_at,
            next_debit_at=next_debit_at,
            pre_debit_notified=pre_debit_notified,
            created_at=created_at,
            updated_at=updated_at,
        )

        mandate.additional_properties = d
        return mandate

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
