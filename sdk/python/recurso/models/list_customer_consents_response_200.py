from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, Literal, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.consent import Consent


T = TypeVar("T", bound="ListCustomerConsentsResponse200")


@_attrs_define
class ListCustomerConsentsResponse200:
    """
    Attributes:
        object_ (Literal['list'] | Unset):
        data (list[Consent] | Unset):
    """

    object_: Literal["list"] | Unset = UNSET
    data: list[Consent] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        object_ = self.object_

        data: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = []
            for data_item_data in self.data:
                data_item = data_item_data.to_dict()
                data.append(data_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if object_ is not UNSET:
            field_dict["object"] = object_
        if data is not UNSET:
            field_dict["data"] = data

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.consent import Consent

        d = dict(src_dict)
        object_ = cast(Literal["list"] | Unset, d.pop("object", UNSET))
        if object_ != "list" and not isinstance(object_, Unset):
            raise ValueError(f"object must match const 'list', got '{object_}'")

        _data = d.pop("data", UNSET)
        data: list[Consent] | Unset = UNSET
        if _data is not UNSET:
            data = []
            for data_item_data in _data:
                data_item = Consent.from_dict(data_item_data)

                data.append(data_item)

        list_customer_consents_response_200 = cls(
            object_=object_,
            data=data,
        )

        list_customer_consents_response_200.additional_properties = d
        return list_customer_consents_response_200

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
