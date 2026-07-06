from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.churn_score_result_risk_level import ChurnScoreResultRiskLevel
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.churn_features import ChurnFeatures


T = TypeVar("T", bound="ChurnScoreResult")


@_attrs_define
class ChurnScoreResult:
    """
    Attributes:
        customer_id (UUID | Unset):
        score (int | Unset):
        risk_level (ChurnScoreResultRiskLevel | Unset):
        features (ChurnFeatures | Unset):
        model_version (str | Unset):
    """

    customer_id: UUID | Unset = UNSET
    score: int | Unset = UNSET
    risk_level: ChurnScoreResultRiskLevel | Unset = UNSET
    features: ChurnFeatures | Unset = UNSET
    model_version: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        customer_id: str | Unset = UNSET
        if not isinstance(self.customer_id, Unset):
            customer_id = str(self.customer_id)

        score = self.score

        risk_level: str | Unset = UNSET
        if not isinstance(self.risk_level, Unset):
            risk_level = self.risk_level.value

        features: dict[str, Any] | Unset = UNSET
        if not isinstance(self.features, Unset):
            features = self.features.to_dict()

        model_version = self.model_version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if customer_id is not UNSET:
            field_dict["customer_id"] = customer_id
        if score is not UNSET:
            field_dict["score"] = score
        if risk_level is not UNSET:
            field_dict["risk_level"] = risk_level
        if features is not UNSET:
            field_dict["features"] = features
        if model_version is not UNSET:
            field_dict["model_version"] = model_version

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.churn_features import ChurnFeatures

        d = dict(src_dict)
        _customer_id = d.pop("customer_id", UNSET)
        customer_id: UUID | Unset
        if isinstance(_customer_id, Unset):
            customer_id = UNSET
        else:
            customer_id = UUID(_customer_id)

        score = d.pop("score", UNSET)

        _risk_level = d.pop("risk_level", UNSET)
        risk_level: ChurnScoreResultRiskLevel | Unset
        if isinstance(_risk_level, Unset):
            risk_level = UNSET
        else:
            risk_level = ChurnScoreResultRiskLevel(_risk_level)

        _features = d.pop("features", UNSET)
        features: ChurnFeatures | Unset
        if isinstance(_features, Unset):
            features = UNSET
        else:
            features = ChurnFeatures.from_dict(_features)

        model_version = d.pop("model_version", UNSET)

        churn_score_result = cls(
            customer_id=customer_id,
            score=score,
            risk_level=risk_level,
            features=features,
            model_version=model_version,
        )

        churn_score_result.additional_properties = d
        return churn_score_result

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
