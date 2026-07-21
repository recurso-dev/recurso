package email

// PreChargeEmailData contains data for pre-charge reminder emails
type PreChargeEmailData struct {
	CustomerName  string
	CustomerEmail string
	PlanName      string
	Amount        string
	ChargeDate    string
	PaymentMethod string
	PortalURL     string
}

// DunningEmailData contains data for dunning emails
type DunningEmailData struct {
	CustomerName     string
	CustomerEmail    string
	InvoiceNumber    string
	Amount           string
	DaysOverdue      int
	RetryCount       int
	NextRetryDate    string
	SuspensionDate   string
	PayNowURL        string
	UpdatePaymentURL string
}

// TrialEndingEmailData contains data for trial-ending reminder emails
type TrialEndingEmailData struct {
	CustomerName  string
	CustomerEmail string
	PlanName      string
	Amount        string // first charge after the trial
	TrialEndDate  string // e.g. "July 14, 2026"
	PortalURL     string
}

// CardExpiringEmailData contains data for card expiry warning emails
type CardExpiringEmailData struct {
	CustomerName     string
	CustomerEmail    string
	CardBrand        string
	CardLast4        string
	ExpiryDate       string // e.g. "July 2026"
	UpdatePaymentURL string
}

// NexusAlertData drives the US economic-nexus threshold alert email (Track D · D1).
// The recipient is the tenant's owner/admin, not a customer.
type NexusAlertData struct {
	RecipientEmail string
	RecipientName  string
	State          string // state code, e.g. "CA"
	Level          string // "approaching" | "crossed"
	ProximityPct   int    // e.g. 82
	TaxableSales   string // formatted USD, e.g. "$92,400.00"
	TxnCount       int
	ThresholdText  string // e.g. "$100,000.00 or 200 transactions"
	SettingsURL    string
}
