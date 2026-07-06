from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.mandate import Mandate


T = TypeVar("T", bound="CreateMandateResponse201")


@_attrs_define
class CreateMandateResponse201:
    """
    Attributes:
        mandate (Mandate | Unset):
        auth_url (str | Unset): URL where the customer authorizes the mandate.
    """

    mandate: Mandate | Unset = UNSET
    auth_url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        mandate: dict[str, Any] | Unset = UNSET
        if not isinstance(self.mandate, Unset):
            mandate = self.mandate.to_dict()

        auth_url = self.auth_url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if mandate is not UNSET:
            field_dict["mandate"] = mandate
        if auth_url is not UNSET:
            field_dict["auth_url"] = auth_url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mandate import Mandate

        d = dict(src_dict)
        _mandate = d.pop("mandate", UNSET)
        mandate: Mandate | Unset
        if isinstance(_mandate, Unset):
            mandate = UNSET
        else:
            mandate = Mandate.from_dict(_mandate)

        auth_url = d.pop("auth_url", UNSET)

        create_mandate_response_201 = cls(
            mandate=mandate,
            auth_url=auth_url,
        )

        create_mandate_response_201.additional_properties = d
        return create_mandate_response_201

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
