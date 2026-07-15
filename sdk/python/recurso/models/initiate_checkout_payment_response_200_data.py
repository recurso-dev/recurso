from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.initiate_checkout_payment_response_200_data_gateway import InitiateCheckoutPaymentResponse200DataGateway
from ..types import UNSET, Unset

T = TypeVar("T", bound="InitiateCheckoutPaymentResponse200Data")


@_attrs_define
class InitiateCheckoutPaymentResponse200Data:
    """
    Attributes:
        order_id (str | Unset): Gateway order/intent id (Stripe `pi_*`, Razorpay `order_*`).
        amount (int | Unset): Amount in the lowest currency unit.
        currency (str | Unset):
        invoice_id (UUID | Unset):
        invoice_number (str | Unset):
        gateway (InitiateCheckoutPaymentResponse200DataGateway | Unset): Which client-side flow to use.
        client_secret (str | Unset): Stripe PaymentIntent client secret (empty for non-Stripe gateways).
        publishable_key (str | Unset): Stripe publishable key for mounting the Payment Element.
        razorpay_key_id (str | Unset): Razorpay public key id for Checkout.js.
    """

    order_id: str | Unset = UNSET
    amount: int | Unset = UNSET
    currency: str | Unset = UNSET
    invoice_id: UUID | Unset = UNSET
    invoice_number: str | Unset = UNSET
    gateway: InitiateCheckoutPaymentResponse200DataGateway | Unset = UNSET
    client_secret: str | Unset = UNSET
    publishable_key: str | Unset = UNSET
    razorpay_key_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        order_id = self.order_id

        amount = self.amount

        currency = self.currency

        invoice_id: str | Unset = UNSET
        if not isinstance(self.invoice_id, Unset):
            invoice_id = str(self.invoice_id)

        invoice_number = self.invoice_number

        gateway: str | Unset = UNSET
        if not isinstance(self.gateway, Unset):
            gateway = self.gateway.value

        client_secret = self.client_secret

        publishable_key = self.publishable_key

        razorpay_key_id = self.razorpay_key_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if order_id is not UNSET:
            field_dict["order_id"] = order_id
        if amount is not UNSET:
            field_dict["amount"] = amount
        if currency is not UNSET:
            field_dict["currency"] = currency
        if invoice_id is not UNSET:
            field_dict["invoice_id"] = invoice_id
        if invoice_number is not UNSET:
            field_dict["invoice_number"] = invoice_number
        if gateway is not UNSET:
            field_dict["gateway"] = gateway
        if client_secret is not UNSET:
            field_dict["client_secret"] = client_secret
        if publishable_key is not UNSET:
            field_dict["publishable_key"] = publishable_key
        if razorpay_key_id is not UNSET:
            field_dict["razorpay_key_id"] = razorpay_key_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        order_id = d.pop("order_id", UNSET)

        amount = d.pop("amount", UNSET)

        currency = d.pop("currency", UNSET)

        _invoice_id = d.pop("invoice_id", UNSET)
        invoice_id: UUID | Unset
        if isinstance(_invoice_id, Unset):
            invoice_id = UNSET
        else:
            invoice_id = UUID(_invoice_id)

        invoice_number = d.pop("invoice_number", UNSET)

        _gateway = d.pop("gateway", UNSET)
        gateway: InitiateCheckoutPaymentResponse200DataGateway | Unset
        if isinstance(_gateway, Unset):
            gateway = UNSET
        else:
            gateway = InitiateCheckoutPaymentResponse200DataGateway(_gateway)

        client_secret = d.pop("client_secret", UNSET)

        publishable_key = d.pop("publishable_key", UNSET)

        razorpay_key_id = d.pop("razorpay_key_id", UNSET)

        initiate_checkout_payment_response_200_data = cls(
            order_id=order_id,
            amount=amount,
            currency=currency,
            invoice_id=invoice_id,
            invoice_number=invoice_number,
            gateway=gateway,
            client_secret=client_secret,
            publishable_key=publishable_key,
            razorpay_key_id=razorpay_key_id,
        )

        initiate_checkout_payment_response_200_data.additional_properties = d
        return initiate_checkout_payment_response_200_data

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
