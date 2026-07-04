package gsp

import (
	"fmt"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// GST INV-01 JSON schema structs for NIC e-invoice API

type GSTInvoiceSchema struct {
	Version  string      `json:"Version"`
	TranDtls TranDtls    `json:"TranDtls"`
	DocDtls  DocDtls     `json:"DocDtls"`
	SellerDtls SellerDtls `json:"SellerDtls"`
	BuyerDtls  BuyerDtls  `json:"BuyerDtls"`
	ItemList   []ItemDtls `json:"ItemList"`
	ValDtls    ValDtls    `json:"ValDtls"`
}

type TranDtls struct {
	TaxSch  string `json:"TaxSch"`  // "GST"
	SupTyp  string `json:"SupTyp"`  // "B2B", "SEZWP", "SEZWOP", "EXPWP", "EXPWOP", "DEXP"
	RegRev  string `json:"RegRev"`  // "Y" or "N" - reverse charge
	IgstOnIntra string `json:"IgstOnIntra"` // "Y" or "N"
}

type DocDtls struct {
	Typ string `json:"Typ"` // "INV", "CRN", "DBN"
	No  string `json:"No"`
	Dt  string `json:"Dt"` // dd/mm/yyyy format
}

type SellerDtls struct {
	Gstin  string `json:"Gstin"`
	LglNm  string `json:"LglNm"`
	TrdNm  string `json:"TrdNm,omitempty"`
	Addr1  string `json:"Addr1"`
	Addr2  string `json:"Addr2,omitempty"`
	Loc    string `json:"Loc"`
	Pin    int    `json:"Pin"`
	Stcd   string `json:"Stcd"`
}

type BuyerDtls struct {
	Gstin  string `json:"Gstin"`
	LglNm  string `json:"LglNm"`
	TrdNm  string `json:"TrdNm,omitempty"`
	Addr1  string `json:"Addr1"`
	Addr2  string `json:"Addr2,omitempty"`
	Loc    string `json:"Loc"`
	Pin    int    `json:"Pin"`
	Pos    string `json:"Pos"` // Place of supply state code
	Stcd   string `json:"Stcd"`
}

type ItemDtls struct {
	SlNo    string  `json:"SlNo"`
	PrdDesc string  `json:"PrdDesc"`
	IsServc string  `json:"IsServc"` // "Y" for service, "N" for goods
	HsnCd   string  `json:"HsnCd"`
	Qty     float64 `json:"Qty,omitempty"`
	Unit    string  `json:"Unit,omitempty"`
	UnitPrice float64 `json:"UnitPrice"`
	TotAmt  float64 `json:"TotAmt"`
	AssAmt  float64 `json:"AssAmt"` // Assessable amount (taxable value)
	GstRt   float64 `json:"GstRt"`
	IgstAmt float64 `json:"IgstAmt"`
	CgstAmt float64 `json:"CgstAmt"`
	SgstAmt float64 `json:"SgstAmt"`
	TotItemVal float64 `json:"TotItemVal"`
}

type ValDtls struct {
	AssVal  float64 `json:"AssVal"`  // Total assessable value
	IgstVal float64 `json:"IgstVal"` // Total IGST
	CgstVal float64 `json:"CgstVal"` // Total CGST
	SgstVal float64 `json:"SgstVal"` // Total SGST
	TotInvVal float64 `json:"TotInvVal"` // Total invoice value
}

// BuildInvoiceSchema maps an EInvoiceRequest to the NIC GST INV-01 JSON schema.
func BuildInvoiceSchema(req *port.EInvoiceRequest) *GSTInvoiceSchema {
	inv := req.Invoice

	// Determine supply type based on seller/buyer state codes
	supTyp := "B2B"
	isInterState := req.Seller.StateCode != req.Buyer.StateCode

	// Format date as dd/mm/yyyy
	docDate := inv.CreatedAt.Format("02/01/2006")

	// Convert amounts from paise to rupees (float)
	toRupees := func(paise int64) float64 {
		return float64(paise) / 100.0
	}

	schema := &GSTInvoiceSchema{
		Version: "1.1",
		TranDtls: TranDtls{
			TaxSch:      "GST",
			SupTyp:      supTyp,
			RegRev:      "N",
			IgstOnIntra: "N",
		},
		DocDtls: DocDtls{
			Typ: "INV",
			No:  inv.InvoiceNumber,
			Dt:  docDate,
		},
		SellerDtls: SellerDtls{
			Gstin: req.Seller.GSTIN,
			LglNm: req.Seller.LegalName,
			TrdNm: req.Seller.TradeName,
			Addr1: req.Seller.Address,
			Loc:   req.Seller.Location,
			Pin:   0, // Will be parsed from PinCode
			Stcd:  req.Seller.StateCode,
		},
		BuyerDtls: BuyerDtls{
			Gstin: req.Buyer.GSTIN,
			LglNm: req.Buyer.LegalName,
			TrdNm: req.Buyer.TradeName,
			Addr1: req.Buyer.Address,
			Loc:   req.Buyer.Location,
			Pin:   0,
			Pos:   req.Buyer.Place,
			Stcd:  req.Buyer.StateCode,
		},
		ValDtls: ValDtls{
			AssVal:    toRupees(inv.Subtotal),
			IgstVal:   toRupees(inv.IGSTAmount),
			CgstVal:   toRupees(inv.CGSTAmount),
			SgstVal:   toRupees(inv.SGSTAmount),
			TotInvVal: toRupees(inv.Total),
		},
	}

	// Build item list
	if len(req.Items) > 0 {
		for _, item := range req.Items {
			schema.ItemList = append(schema.ItemList, ItemDtls{
				SlNo:       fmt.Sprintf("%d", item.SlNo),
				PrdDesc:    item.Description,
				IsServc:    "Y", // SaaS is a service
				HsnCd:      item.HSNCode,
				Qty:        item.Quantity,
				Unit:       item.Unit,
				UnitPrice:  toRupees(item.UnitPrice),
				TotAmt:     toRupees(item.TotalAmount),
				AssAmt:     toRupees(item.TotalAmount),
				GstRt:      item.TaxRate,
				IgstAmt:    toRupees(item.IGSTAmount),
				CgstAmt:    toRupees(item.CGSTAmount),
				SgstAmt:    toRupees(item.SGSTAmount),
				TotItemVal: toRupees(item.TotalAmount + item.IGSTAmount + item.CGSTAmount + item.SGSTAmount),
			})
		}
	} else {
		// Single line item from invoice totals
		taxRate := 18.0 // Default GST rate
		var igstAmt, cgstAmt, sgstAmt float64
		if isInterState {
			igstAmt = toRupees(inv.IGSTAmount)
		} else {
			cgstAmt = toRupees(inv.CGSTAmount)
			sgstAmt = toRupees(inv.SGSTAmount)
		}

		schema.ItemList = []ItemDtls{
			{
				SlNo:       "1",
				PrdDesc:    "SaaS Subscription",
				IsServc:    "Y",
				HsnCd:      "998314",
				Qty:        1,
				Unit:       "NOS",
				UnitPrice:  toRupees(inv.Subtotal),
				TotAmt:     toRupees(inv.Subtotal),
				AssAmt:     toRupees(inv.Subtotal),
				GstRt:      taxRate,
				IgstAmt:    igstAmt,
				CgstAmt:    cgstAmt,
				SgstAmt:    sgstAmt,
				TotItemVal: toRupees(inv.Total),
			},
		}
	}

	_ = time.Now // avoid unused import

	return schema
}
