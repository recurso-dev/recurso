package service

import (
	"fmt"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// This file maps the internal GSTR1Return (minor units, readable field names)
// to the government GSTN GSTR-1 JSON structure: official field names, amounts
// in rupees. Validate the output against the GSTN GSTR-1 schema / Returns
// Offline Tool before filing — that is the objective format gate.

// GSTR1GovDocument is the top-level GSTN GSTR-1 JSON.
type GSTR1GovDocument struct {
	GSTIN string    `json:"gstin"`
	FP    string    `json:"fp"` // filing period, MMYYYY
	B2B   []govB2B  `json:"b2b,omitempty"`
	B2CS  []govB2CS `json:"b2cs,omitempty"`
	CDNR  []govCDNR `json:"cdnr,omitempty"`
	HSN   *govHSN   `json:"hsn,omitempty"`
}

type govItemDetail struct {
	Rate   float64 `json:"rt"`
	TxVal  float64 `json:"txval"`
	IAmt   float64 `json:"iamt"`
	CAmt   float64 `json:"camt"`
	SAmt   float64 `json:"samt"`
	CesAmt float64 `json:"csamt"`
}

type govItem struct {
	Num    int           `json:"num"`
	ItmDet govItemDetail `json:"itm_det"`
}

type govB2BInvoice struct {
	INum   string    `json:"inum"`
	IDt    string    `json:"idt"` // DD-MM-YYYY
	Val    float64   `json:"val"`
	Pos    string    `json:"pos"`
	Rchrg  string    `json:"rchrg"`   // "N" — reverse charge not modelled in v1
	InvTyp string    `json:"inv_typ"` // "R" — regular
	Items  []govItem `json:"itms"`
}

type govB2B struct {
	Ctin string          `json:"ctin"`
	Inv  []govB2BInvoice `json:"inv"`
}

type govB2CS struct {
	SplyTy string  `json:"sply_ty"` // INTRA | INTER
	Pos    string  `json:"pos"`
	Typ    string  `json:"typ"` // "OE" (other than e-commerce)
	Rate   float64 `json:"rt"`
	TxVal  float64 `json:"txval"`
	IAmt   float64 `json:"iamt"`
	CAmt   float64 `json:"camt"`
	SAmt   float64 `json:"samt"`
	CesAmt float64 `json:"csamt"`
}

type govCDNRNote struct {
	NtTy   string    `json:"ntty"` // "C" credit / "D" debit
	NtNum  string    `json:"nt_num"`
	NtDt   string    `json:"nt_dt"`
	Val    float64   `json:"val"`
	Pos    string    `json:"pos"`
	Rchrg  string    `json:"rchrg"`
	InvTyp string    `json:"inv_typ"`
	Items  []govItem `json:"itms"`
}

type govCDNR struct {
	Ctin string        `json:"ctin"`
	Nt   []govCDNRNote `json:"nt"`
}

type govHSNRow struct {
	Num    int     `json:"num"`
	HsnSc  string  `json:"hsn_sc"`
	Desc   string  `json:"desc"`
	Uqc    string  `json:"uqc"`
	Qty    float64 `json:"qty"`
	TxVal  float64 `json:"txval"`
	IAmt   float64 `json:"iamt"`
	CAmt   float64 `json:"camt"`
	SAmt   float64 `json:"samt"`
	CesAmt float64 `json:"csamt"`
}

type govHSN struct {
	Data []govHSNRow `json:"data"`
}

func rupees(paise int64) float64 { return float64(paise) / 100.0 }

func supplyType(igst int64) string {
	if igst > 0 {
		return "INTER"
	}
	return "INTRA"
}

// BuildGSTR1GovDocument maps the internal return to the GSTN GSTR-1 JSON schema
// for the seller's GSTIN. Amounts become rupees; the item detail sums
// (txval + tax) reconcile to the invoice value that the tool validates.
func BuildGSTR1GovDocument(sellerGSTIN string, r *domain.GSTR1Return) *GSTR1GovDocument {
	doc := &GSTR1GovDocument{
		GSTIN: sellerGSTIN,
		FP:    fmt.Sprintf("%02d%04d", r.Month, r.Year),
	}

	for _, g := range r.B2B {
		gg := govB2B{Ctin: g.GSTIN}
		for _, inv := range g.Invoices {
			gg.Inv = append(gg.Inv, govB2BInvoice{
				INum:   inv.InvoiceNumber,
				IDt:    inv.Date.Format("02-01-2006"),
				Val:    rupees(inv.TaxableValue + inv.IGST + inv.CGST + inv.SGST),
				Pos:    inv.PlaceOfSupply,
				Rchrg:  "N",
				InvTyp: "R",
				Items: []govItem{{Num: 1, ItmDet: govItemDetail{
					Rate: inv.Rate, TxVal: rupees(inv.TaxableValue),
					IAmt: rupees(inv.IGST), CAmt: rupees(inv.CGST), SAmt: rupees(inv.SGST),
				}}},
			})
		}
		doc.B2B = append(doc.B2B, gg)
	}

	for _, c := range r.B2CS {
		doc.B2CS = append(doc.B2CS, govB2CS{
			SplyTy: supplyType(c.IGST), Pos: c.PlaceOfSupply, Typ: "OE", Rate: c.Rate,
			TxVal: rupees(c.TaxableValue), IAmt: rupees(c.IGST), CAmt: rupees(c.CGST), SAmt: rupees(c.SGST),
		})
	}

	for _, g := range r.CDNR {
		gg := govCDNR{Ctin: g.GSTIN}
		for _, n := range g.Notes {
			gg.Nt = append(gg.Nt, govCDNRNote{
				NtTy: "C", NtNum: n.NoteNumber, NtDt: n.Date.Format("02-01-2006"),
				Val: rupees(n.TaxableValue + n.IGST + n.CGST + n.SGST), Pos: n.PlaceOfSupply, Rchrg: "N", InvTyp: "R",
				Items: []govItem{{Num: 1, ItmDet: govItemDetail{
					Rate: n.Rate, TxVal: rupees(n.TaxableValue),
					IAmt: rupees(n.IGST), CAmt: rupees(n.CGST), SAmt: rupees(n.SGST),
				}}},
			})
		}
		doc.CDNR = append(doc.CDNR, gg)
	}

	if len(r.HSN) > 0 {
		h := &govHSN{}
		for i, row := range r.HSN {
			h.Data = append(h.Data, govHSNRow{
				Num: i + 1, HsnSc: row.HSNCode, Uqc: "OTH",
				TxVal: rupees(row.TaxableValue), IAmt: rupees(row.IGST), CAmt: rupees(row.CGST), SAmt: rupees(row.SGST),
			})
		}
		doc.HSN = h
	}

	return doc
}
