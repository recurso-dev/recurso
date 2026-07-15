from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.tax_nexus_nexus_type import TaxNexusNexusType
from ..types import UNSET, Unset

T = TypeVar("T", bound="TaxNexus")


@_attrs_define
class TaxNexus:
    """
    Attributes:
        state_code (str | Unset):  Example: CA.
        nexus_type (TaxNexusNexusType | Unset):
        established_at (datetime.datetime | None | Unset):
        created_at (datetime.datetime | Unset):
    """

    state_code: str | Unset = UNSET
    nexus_type: TaxNexusNexusType | Unset = UNSET
    established_at: datetime.datetime | None | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        state_code = self.state_code

        nexus_type: str | Unset = UNSET
        if not isinstance(self.nexus_type, Unset):
            nexus_type = self.nexus_type.value

        established_at: None | str | Unset
        if isinstance(self.established_at, Unset):
            established_at = UNSET
        elif isinstance(self.established_at, datetime.datetime):
            established_at = self.established_at.isoformat()
        else:
            established_at = self.established_at

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if state_code is not UNSET:
            field_dict["state_code"] = state_code
        if nexus_type is not UNSET:
            field_dict["nexus_type"] = nexus_type
        if established_at is not UNSET:
            field_dict["established_at"] = established_at
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state_code = d.pop("state_code", UNSET)

        _nexus_type = d.pop("nexus_type", UNSET)
        nexus_type: TaxNexusNexusType | Unset
        if isinstance(_nexus_type, Unset):
            nexus_type = UNSET
        else:
            nexus_type = TaxNexusNexusType(_nexus_type)

        def _parse_established_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                established_at_type_0 = datetime.datetime.fromisoformat(data)

                return established_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        established_at = _parse_established_at(d.pop("established_at", UNSET))

        _created_at = d.pop("created_at", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        tax_nexus = cls(
            state_code=state_code,
            nexus_type=nexus_type,
            established_at=established_at,
            created_at=created_at,
        )

        tax_nexus.additional_properties = d
        return tax_nexus

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
