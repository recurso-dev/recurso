from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.portal_confirm_payment_method_response_200_data_status import (
    PortalConfirmPaymentMethodResponse200DataStatus,
)
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.portal_confirm_payment_method_response_200_data_card import (
        PortalConfirmPaymentMethodResponse200DataCard,
    )


T = TypeVar("T", bound="PortalConfirmPaymentMethodResponse200Data")


@_attrs_define
class PortalConfirmPaymentMethodResponse200Data:
    """
    Attributes:
        status (PortalConfirmPaymentMethodResponse200DataStatus | Unset):
        card (PortalConfirmPaymentMethodResponse200DataCard | Unset):
    """

    status: PortalConfirmPaymentMethodResponse200DataStatus | Unset = UNSET
    card: PortalConfirmPaymentMethodResponse200DataCard | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value

        card: dict[str, Any] | Unset = UNSET
        if not isinstance(self.card, Unset):
            card = self.card.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if status is not UNSET:
            field_dict["status"] = status
        if card is not UNSET:
            field_dict["card"] = card

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.portal_confirm_payment_method_response_200_data_card import (
            PortalConfirmPaymentMethodResponse200DataCard,
        )

        d = dict(src_dict)
        _status = d.pop("status", UNSET)
        status: PortalConfirmPaymentMethodResponse200DataStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = PortalConfirmPaymentMethodResponse200DataStatus(_status)

        _card = d.pop("card", UNSET)
        card: PortalConfirmPaymentMethodResponse200DataCard | Unset
        if isinstance(_card, Unset):
            card = UNSET
        else:
            card = PortalConfirmPaymentMethodResponse200DataCard.from_dict(_card)

        portal_confirm_payment_method_response_200_data = cls(
            status=status,
            card=card,
        )

        portal_confirm_payment_method_response_200_data.additional_properties = d
        return portal_confirm_payment_method_response_200_data

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
