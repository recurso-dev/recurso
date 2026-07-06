from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.entitlement_kind import EntitlementKind
from ..types import UNSET, Unset

T = TypeVar("T", bound="Entitlement")


@_attrs_define
class Entitlement:
    """
    Attributes:
        id (UUID | Unset):
        plan_id (UUID | Unset):
        feature_key (str | Unset):
        kind (EntitlementKind | Unset):
        bool_value (bool | None | Unset):
        limit_value (int | None | Unset):
    """

    id: UUID | Unset = UNSET
    plan_id: UUID | Unset = UNSET
    feature_key: str | Unset = UNSET
    kind: EntitlementKind | Unset = UNSET
    bool_value: bool | None | Unset = UNSET
    limit_value: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        plan_id: str | Unset = UNSET
        if not isinstance(self.plan_id, Unset):
            plan_id = str(self.plan_id)

        feature_key = self.feature_key

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value

        bool_value: bool | None | Unset
        if isinstance(self.bool_value, Unset):
            bool_value = UNSET
        else:
            bool_value = self.bool_value

        limit_value: int | None | Unset
        if isinstance(self.limit_value, Unset):
            limit_value = UNSET
        else:
            limit_value = self.limit_value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if id is not UNSET:
            field_dict["id"] = id
        if plan_id is not UNSET:
            field_dict["plan_id"] = plan_id
        if feature_key is not UNSET:
            field_dict["feature_key"] = feature_key
        if kind is not UNSET:
            field_dict["kind"] = kind
        if bool_value is not UNSET:
            field_dict["bool_value"] = bool_value
        if limit_value is not UNSET:
            field_dict["limit_value"] = limit_value

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

        _plan_id = d.pop("plan_id", UNSET)
        plan_id: UUID | Unset
        if isinstance(_plan_id, Unset):
            plan_id = UNSET
        else:
            plan_id = UUID(_plan_id)

        feature_key = d.pop("feature_key", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: EntitlementKind | Unset
        if isinstance(_kind, Unset):
            kind = UNSET
        else:
            kind = EntitlementKind(_kind)

        def _parse_bool_value(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        bool_value = _parse_bool_value(d.pop("bool_value", UNSET))

        def _parse_limit_value(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        limit_value = _parse_limit_value(d.pop("limit_value", UNSET))

        entitlement = cls(
            id=id,
            plan_id=plan_id,
            feature_key=feature_key,
            kind=kind,
            bool_value=bool_value,
            limit_value=limit_value,
        )

        entitlement.additional_properties = d
        return entitlement

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
