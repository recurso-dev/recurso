from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.get_customer_entitlements_response_200_data_item_kind import (
    GetCustomerEntitlementsResponse200DataItemKind,
)
from ..types import UNSET, Unset

T = TypeVar("T", bound="GetCustomerEntitlementsResponse200DataItem")


@_attrs_define
class GetCustomerEntitlementsResponse200DataItem:
    """
    Attributes:
        feature_key (str | Unset):
        kind (GetCustomerEntitlementsResponse200DataItemKind | Unset):
        bool_value (bool | None | Unset):
        limit_value (int | None | Unset):
        plan_ids (list[UUID] | Unset):
    """

    feature_key: str | Unset = UNSET
    kind: GetCustomerEntitlementsResponse200DataItemKind | Unset = UNSET
    bool_value: bool | None | Unset = UNSET
    limit_value: int | None | Unset = UNSET
    plan_ids: list[UUID] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
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

        plan_ids: list[str] | Unset = UNSET
        if not isinstance(self.plan_ids, Unset):
            plan_ids = []
            for plan_ids_item_data in self.plan_ids:
                plan_ids_item = str(plan_ids_item_data)
                plan_ids.append(plan_ids_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if feature_key is not UNSET:
            field_dict["feature_key"] = feature_key
        if kind is not UNSET:
            field_dict["kind"] = kind
        if bool_value is not UNSET:
            field_dict["bool_value"] = bool_value
        if limit_value is not UNSET:
            field_dict["limit_value"] = limit_value
        if plan_ids is not UNSET:
            field_dict["plan_ids"] = plan_ids

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        feature_key = d.pop("feature_key", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: GetCustomerEntitlementsResponse200DataItemKind | Unset
        if isinstance(_kind, Unset):
            kind = UNSET
        else:
            kind = GetCustomerEntitlementsResponse200DataItemKind(_kind)

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

        _plan_ids = d.pop("plan_ids", UNSET)
        plan_ids: list[UUID] | Unset = UNSET
        if _plan_ids is not UNSET:
            plan_ids = []
            for plan_ids_item_data in _plan_ids:
                plan_ids_item = UUID(plan_ids_item_data)

                plan_ids.append(plan_ids_item)

        get_customer_entitlements_response_200_data_item = cls(
            feature_key=feature_key,
            kind=kind,
            bool_value=bool_value,
            limit_value=limit_value,
            plan_ids=plan_ids,
        )

        get_customer_entitlements_response_200_data_item.additional_properties = d
        return get_customer_entitlements_response_200_data_item

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
