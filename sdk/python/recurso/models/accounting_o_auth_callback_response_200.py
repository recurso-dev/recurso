from __future__ import annotations

from collections.abc import Mapping
from typing import Any, Literal, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AccountingOAuthCallbackResponse200")


@_attrs_define
class AccountingOAuthCallbackResponse200:
    """
    Attributes:
        status (Literal['connected'] | Unset):
        connection_id (UUID | Unset):
    """

    status: Literal["connected"] | Unset = UNSET
    connection_id: UUID | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status = self.status

        connection_id: str | Unset = UNSET
        if not isinstance(self.connection_id, Unset):
            connection_id = str(self.connection_id)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if status is not UNSET:
            field_dict["status"] = status
        if connection_id is not UNSET:
            field_dict["connection_id"] = connection_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = cast(Literal["connected"] | Unset, d.pop("status", UNSET))
        if status != "connected" and not isinstance(status, Unset):
            raise ValueError(f"status must match const 'connected', got '{status}'")

        _connection_id = d.pop("connection_id", UNSET)
        connection_id: UUID | Unset
        if isinstance(_connection_id, Unset):
            connection_id = UNSET
        else:
            connection_id = UUID(_connection_id)

        accounting_o_auth_callback_response_200 = cls(
            status=status,
            connection_id=connection_id,
        )

        accounting_o_auth_callback_response_200.additional_properties = d
        return accounting_o_auth_callback_response_200

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
