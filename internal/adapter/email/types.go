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
