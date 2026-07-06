from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.get_portal_data_response_200_customer import GetPortalDataResponse200Customer
    from ..models.invoice import Invoice
    from ..models.subscription import Subscription


T = TypeVar("T", bound="GetPortalDataResponse200")


@_attrs_define
class GetPortalDataResponse200:
    """
    Attributes:
        customer (GetPortalDataResponse200Customer | Unset):
        subscriptions (list[Subscription] | None | Unset):
        invoices (list[Invoice] | None | Unset):
    """

    customer: GetPortalDataResponse200Customer | Unset = UNSET
    subscriptions: list[Subscription] | None | Unset = UNSET
    invoices: list[Invoice] | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        customer: dict[str, Any] | Unset = UNSET
        if not isinstance(self.customer, Unset):
            customer = self.customer.to_dict()

        subscriptions: list[dict[str, Any]] | None | Unset
        if isinstance(self.subscriptions, Unset):
            subscriptions = UNSET
        elif isinstance(self.subscriptions, list):
            subscriptions = []
            for subscriptions_type_0_item_data in self.subscriptions:
                subscriptions_type_0_item = subscriptions_type_0_item_data.to_dict()
                subscriptions.append(subscriptions_type_0_item)

        else:
            subscriptions = self.subscriptions

        invoices: list[dict[str, Any]] | None | Unset
        if isinstance(self.invoices, Unset):
            invoices = UNSET
        elif isinstance(self.invoices, list):
            invoices = []
            for invoices_type_0_item_data in self.invoices:
                invoices_type_0_item = invoices_type_0_item_data.to_dict()
                invoices.append(invoices_type_0_item)

        else:
            invoices = self.invoices

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if customer is not UNSET:
            field_dict["customer"] = customer
        if subscriptions is not UNSET:
            field_dict["subscriptions"] = subscriptions
        if invoices is not UNSET:
            field_dict["invoices"] = invoices

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.get_portal_data_response_200_customer import GetPortalDataResponse200Customer
        from ..models.invoice import Invoice
        from ..models.subscription import Subscription

        d = dict(src_dict)
        _customer = d.pop("customer", UNSET)
        customer: GetPortalDataResponse200Customer | Unset
        if isinstance(_customer, Unset):
            customer = UNSET
        else:
            customer = GetPortalDataResponse200Customer.from_dict(_customer)

        def _parse_subscriptions(data: object) -> list[Subscription] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                subscriptions_type_0 = []
                _subscriptions_type_0 = data
                for subscriptions_type_0_item_data in _subscriptions_type_0:
                    subscriptions_type_0_item = Subscription.from_dict(subscriptions_type_0_item_data)

                    subscriptions_type_0.append(subscriptions_type_0_item)

                return subscriptions_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Subscription] | None | Unset, data)

        subscriptions = _parse_subscriptions(d.pop("subscriptions", UNSET))

        def _parse_invoices(data: object) -> list[Invoice] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                invoices_type_0 = []
                _invoices_type_0 = data
                for invoices_type_0_item_data in _invoices_type_0:
                    invoices_type_0_item = Invoice.from_dict(invoices_type_0_item_data)

                    invoices_type_0.append(invoices_type_0_item)

                return invoices_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Invoice] | None | Unset, data)

        invoices = _parse_invoices(d.pop("invoices", UNSET))

        get_portal_data_response_200 = cls(
            customer=customer,
            subscriptions=subscriptions,
            invoices=invoices,
        )

        get_portal_data_response_200.additional_properties = d
        return get_portal_data_response_200

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
