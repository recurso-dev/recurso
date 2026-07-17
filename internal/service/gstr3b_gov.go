package service

import (
	"fmt"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// This file maps the internal GSTR3BReturn (minor units, readable field names)
// to the government GSTN GSTR-3B JSON structure: official field names, amounts
// in rupees. Validate the output against the GSTN GSTR-3B schema / Returns
// Offline Tool before filing — that is the objective format gate.

// GSTR3BGovDocument is the top-level GSTN GSTR-3B JSON.
type GSTR3BGovDocument struct {
	GSTIN      string        `json:"gstin"`
	RetPeriod  string        `json:"ret_period"` // MMYYYY
	SupDetails govSupDetails `json:"sup_details"`
	InterSup   govInterSup   `json:"inter_sup"`
	ITCElg     govITCElg     `json:"itc_elg"`
}

// govSupRow is one Table 3.1 row in government field names (rupees).
type govSupRow struct {
	TxVal  float64 `json:"txval"`
	IAmt   float64 `json:"iamt"`
	CAmt   float64 `json:"camt"`
	SAmt   float64 `json:"samt"`
	CesAmt float64 `json:"csamt"`
}

// govSupDetails is Table 3.1 — outward supplies and inward reverse-charge.
type govSupDetails struct {
	OSupDet     govSupRow `json:"osup_det"`      // 3.1(a) taxable outward
	OSupZero    govSupRow `json:"osup_zero"`     // 3.1(b) zero-rated
	OSupNilExmp govSupRow `json:"osup_nil_exmp"` // 3.1(c) nil/exempt
	ISupRev     govSupRow `json:"isup_rev"`      // 3.1(d) inward reverse charge
	OSupNonGST  govSupRow `json:"osup_nongst"`   // 3.1(e) non-GST outward
}

// govInterSupRow is one Table 3.2 row (place of supply + IGST share).
type govInterSupRow struct {
	Pos   string  `json:"pos"`
	TxVal float64 `json:"txval"`
	IAmt  float64 `json:"iamt"`
}

// govInterSup is Table 3.2 — inter-state supplies to unregistered persons,
// composition taxable persons, and UIN holders. This engine only bills
// regular customers, so composition/UIN stay empty.
type govInterSup struct {
	UnregDetails []govInterSupRow `json:"unreg_details"`
	CompDetails  []govInterSupRow `json:"comp_details"`
	UINDetails   []govInterSupRow `json:"uin_details"`
}

// govITCRow is one ITC value row (all-zero in this export — see govITCElg).
type govITCRow struct {
	IAmt   float64 `json:"iamt"`
	CAmt   float64 `json:"camt"`
	SAmt   float64 `json:"samt"`
	CesAmt float64 `json:"csamt"`
}

// govITCElg is Table 4 (input tax credit). The billing engine holds no
// purchase-side data, so the table is emitted with zero values for schema
// completeness; the taxpayer/CA fills ITC before filing. ITC tracking is
// deferred to P2 per docs/spec_india_decisive.md.
type govITCElg struct {
	ITCNet govITCRow `json:"itc_net"`
}

func supRow(v domain.GSTR3BValues) govSupRow {
	return govSupRow{
		TxVal: rupees(v.TaxableValue),
		IAmt:  rupees(v.IGST),
		CAmt:  rupees(v.CGST),
		SAmt:  rupees(v.SGST),
	}
}

// BuildGSTR3BGovDocument maps the internal return to the GSTN GSTR-3B JSON
// schema for the seller's GSTIN. Amounts become rupees. Table 3.2 slices are
// always non-nil so the JSON carries empty arrays, not null.
func BuildGSTR3BGovDocument(sellerGSTIN string, r *domain.GSTR3BReturn) *GSTR3BGovDocument {
	doc := &GSTR3BGovDocument{
		GSTIN:     sellerGSTIN,
		RetPeriod: fmt.Sprintf("%02d%04d", r.Month, r.Year),
		SupDetails: govSupDetails{
			OSupDet:     supRow(r.OutwardTaxable),
			OSupZero:    supRow(r.ZeroRated),
			OSupNilExmp: supRow(r.NilExempt),
			ISupRev:     supRow(r.InwardReverseCharge),
			OSupNonGST:  supRow(r.NonGST),
		},
		InterSup: govInterSup{
			UnregDetails: []govInterSupRow{},
			CompDetails:  []govInterSupRow{},
			UINDetails:   []govInterSupRow{},
		},
	}

	for _, row := range r.InterStateUnregistered {
		doc.InterSup.UnregDetails = append(doc.InterSup.UnregDetails, govInterSupRow{
			Pos:   row.PlaceOfSupply,
			TxVal: rupees(row.TaxableValue),
			IAmt:  rupees(row.IGST),
		})
	}

	return doc
}
