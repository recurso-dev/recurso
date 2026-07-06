from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.reconciliation_discrepancy import ReconciliationDiscrepancy


T = TypeVar("T", bound="ReconciliationReport")


@_attrs_define
class ReconciliationReport:
    """
    Attributes:
        tenant_id (UUID | Unset):
        started_at (datetime.datetime | Unset):
        finished_at (datetime.datetime | Unset):
        invoices_checked (int | Unset):
        paid_invoices_checked (int | Unset):
        total_discrepancies (int | Unset):
        discrepancies (list[ReconciliationDiscrepancy] | Unset):
        truncated (bool | Unset): True when more discrepancies exist than are listed.
        tb_compared (bool | Unset): Whether TigerBeetle was included in the comparison.
        tb_skip_reason (str | Unset):
    """

    tenant_id: UUID | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    invoices_checked: int | Unset = UNSET
    paid_invoices_checked: int | Unset = UNSET
    total_discrepancies: int | Unset = UNSET
    discrepancies: list[ReconciliationDiscrepancy] | Unset = UNSET
    truncated: bool | Unset = UNSET
    tb_compared: bool | Unset = UNSET
    tb_skip_reason: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        tenant_id: str | Unset = UNSET
        if not isinstance(self.tenant_id, Unset):
            tenant_id = str(self.tenant_id)

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        invoices_checked = self.invoices_checked

        paid_invoices_checked = self.paid_invoices_checked

        total_discrepancies = self.total_discrepancies

        discrepancies: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.discrepancies, Unset):
            discrepancies = []
            for discrepancies_item_data in self.discrepancies:
                discrepancies_item = discrepancies_item_data.to_dict()
                discrepancies.append(discrepancies_item)

        truncated = self.truncated

        tb_compared = self.tb_compared

        tb_skip_reason = self.tb_skip_reason

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if tenant_id is not UNSET:
            field_dict["tenant_id"] = tenant_id
        if started_at is not UNSET:
            field_dict["started_at"] = started_at
        if finished_at is not UNSET:
            field_dict["finished_at"] = finished_at
        if invoices_checked is not UNSET:
            field_dict["invoices_checked"] = invoices_checked
        if paid_invoices_checked is not UNSET:
            field_dict["paid_invoices_checked"] = paid_invoices_checked
        if total_discrepancies is not UNSET:
            field_dict["total_discrepancies"] = total_discrepancies
        if discrepancies is not UNSET:
            field_dict["discrepancies"] = discrepancies
        if truncated is not UNSET:
            field_dict["truncated"] = truncated
        if tb_compared is not UNSET:
            field_dict["tb_compared"] = tb_compared
        if tb_skip_reason is not UNSET:
            field_dict["tb_skip_reason"] = tb_skip_reason

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.reconciliation_discrepancy import ReconciliationDiscrepancy

        d = dict(src_dict)
        _tenant_id = d.pop("tenant_id", UNSET)
        tenant_id: UUID | Unset
        if isinstance(_tenant_id, Unset):
            tenant_id = UNSET
        else:
            tenant_id = UUID(_tenant_id)

        _started_at = d.pop("started_at", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at, Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)

        _finished_at = d.pop("finished_at", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at, Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)

        invoices_checked = d.pop("invoices_checked", UNSET)

        paid_invoices_checked = d.pop("paid_invoices_checked", UNSET)

        total_discrepancies = d.pop("total_discrepancies", UNSET)

        _discrepancies = d.pop("discrepancies", UNSET)
        discrepancies: list[ReconciliationDiscrepancy] | Unset = UNSET
        if _discrepancies is not UNSET:
            discrepancies = []
            for discrepancies_item_data in _discrepancies:
                discrepancies_item = ReconciliationDiscrepancy.from_dict(discrepancies_item_data)

                discrepancies.append(discrepancies_item)

        truncated = d.pop("truncated", UNSET)

        tb_compared = d.pop("tb_compared", UNSET)

        tb_skip_reason = d.pop("tb_skip_reason", UNSET)

        reconciliation_report = cls(
            tenant_id=tenant_id,
            started_at=started_at,
            finished_at=finished_at,
            invoices_checked=invoices_checked,
            paid_invoices_checked=paid_invoices_checked,
            total_discrepancies=total_discrepancies,
            discrepancies=discrepancies,
            truncated=truncated,
            tb_compared=tb_compared,
            tb_skip_reason=tb_skip_reason,
        )

        reconciliation_report.additional_properties = d
        return reconciliation_report

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
