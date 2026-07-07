package email

// EmailBaseTemplate provides a base HTML structure for all emails
const EmailBaseTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Subject}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #334155;
            background-color: #f8fafc;
            margin: 0;
            padding: 0;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 40px 20px;
        }
        .card {
            background: #ffffff;
            border-radius: 12px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            padding: 32px;
            margin-bottom: 24px;
        }
        .logo {
            text-align: center;
            margin-bottom: 24px;
        }
        .logo-icon {
            display: inline-block;
            width: 48px;
            height: 48px;
            background: #0f172a;
            border-radius: 12px;
            color: #ffffff;
            font-size: 24px;
            font-weight: bold;
            line-height: 48px;
            text-align: center;
        }
        h1 {
            color: #0f172a;
            font-size: 24px;
            font-weight: 700;
            margin: 0 0 16px 0;
        }
        h2 {
            color: #334155;
            font-size: 18px;
            font-weight: 600;
            margin: 24px 0 12px 0;
        }
        p {
            margin: 0 0 16px 0;
        }
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background: #0f172a;
            color: #ffffff !important;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            margin: 16px 0;
        }
        .btn:hover {
            background: #1e293b;
        }
        .info-box {
            background: #f1f5f9;
            border-radius: 8px;
            padding: 16px;
            margin: 16px 0;
        }
        .info-row {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #e2e8f0;
        }
        .info-row:last-child {
            border-bottom: none;
        }
        .info-label {
            color: #64748b;
            font-size: 14px;
        }
        .info-value {
            color: #0f172a;
            font-weight: 600;
        }
        .amount {
            font-size: 32px;
            font-weight: 700;
            color: #0f172a;
        }
        .status {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 9999px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
        }
        .status-success { background: #dcfce7; color: #166534; }
        .status-warning { background: #fef3c7; color: #92400e; }
        .status-info { background: #dbeafe; color: #1e40af; }
        .footer {
            text-align: center;
            color: #94a3b8;
            font-size: 12px;
            margin-top: 32px;
        }
        .footer a {
            color: #64748b;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
            <span class="logo-icon">R</span>
        </div>
        <div class="card">
            {{.Content}}
        </div>
        <div class="footer">
            <p>This email was sent by <a href="#">Recurso</a></p>
            <p>If you have questions, please contact support.</p>
        </div>
    </div>
</body>
</html>`

// InvoiceCreatedTemplate for new invoice notifications
const InvoiceCreatedTemplate = `
<h1>New Invoice</h1>
<p>Hello {{.CustomerName}},</p>
<p>A new invoice has been generated for your account.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Invoice Number</span>
        <span class="info-value">{{.InvoiceNumber}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Amount Due</span>
        <span class="info-value">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Due Date</span>
        <span class="info-value">{{.DueDate}}</span>
    </div>
</div>

<p style="text-align: center;">
    <a href="{{.PaymentURL}}" class="btn">Pay Now</a>
</p>

<p>Thank you for your business!</p>
`

// PaymentReceivedTemplate for payment confirmation
const PaymentReceivedTemplate = `
<h1>Payment Received</h1>
<p>Hello {{.CustomerName}},</p>
<p>We've received your payment. Thank you!</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Amount</span>
        <span class="info-value">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Invoice</span>
        <span class="info-value">{{.InvoiceNumber}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Payment Date</span>
        <span class="info-value">{{.PaymentDate}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Status</span>
        <span class="status status-success">PAID</span>
    </div>
</div>

<p>If you have any questions, please don't hesitate to contact us.</p>
`

// SubscriptionCreatedTemplate for new subscription
const SubscriptionCreatedTemplate = `
<h1>Welcome to {{.PlanName}}!</h1>
<p>Hello {{.CustomerName}},</p>
<p>Your subscription has been activated successfully.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Plan</span>
        <span class="info-value">{{.PlanName}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Price</span>
        <span class="info-value">{{.Price}} / {{.Interval}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Started</span>
        <span class="info-value">{{.StartDate}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Next Billing</span>
        <span class="info-value">{{.NextBillingDate}}</span>
    </div>
</div>

<p style="text-align: center;">
    <a href="{{.PortalURL}}" class="btn">Manage Subscription</a>
</p>
`

// MagicLinkTemplate for passwordless login
const MagicLinkTemplate = `
<h1>Login to Your Account</h1>
<p>Hello,</p>
<p>Click the button below to securely log in to your billing portal.</p>

<p style="text-align: center;">
    <a href="{{.LoginURL}}" class="btn">Log In</a>
</p>

<p style="font-size: 14px; color: #64748b;">This link will expire in 15 minutes. If you didn't request this, you can safely ignore this email.</p>
`

// PasswordResetTemplate for admin-dashboard password reset requests.
const PasswordResetTemplate = `
<h1>Reset your password</h1>
<p>Hello,</p>
<p>We received a request to reset the password for your Recurso account. Click the button below to choose a new password.</p>

<p style="text-align: center;">
    <a href="{{.ResetURL}}" class="btn">Reset Password</a>
</p>

<p style="font-size: 14px; color: #64748b;">This link will expire in 1 hour. If you didn't request a password reset, you can safely ignore this email — your password will not change.</p>
`

// PaymentFailedTemplate for failed payments
const PaymentFailedTemplate = `
<h1>Payment Failed</h1>
<p>Hello {{.CustomerName}},</p>
<p>Unfortunately, we were unable to process your payment.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Invoice</span>
        <span class="info-value">{{.InvoiceNumber}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Amount</span>
        <span class="info-value">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Reason</span>
        <span class="info-value">{{.FailureReason}}</span>
    </div>
</div>

<p>Please update your payment method to avoid service interruption.</p>

<p style="text-align: center;">
    <a href="{{.UpdatePaymentURL}}" class="btn">Update Payment Method</a>
</p>
`

// PreChargeReminderTemplate for 24-hour reminder before recurring charge (RBI compliance)
const PreChargeReminderTemplate = `
<h1>Upcoming Payment Reminder</h1>
<p>Hello {{.CustomerName}},</p>
<p>This is a reminder that your subscription will renew in <strong>24 hours</strong>.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Subscription</span>
        <span class="info-value">{{.PlanName}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Amount</span>
        <span class="info-value">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Charge Date</span>
        <span class="info-value">{{.ChargeDate}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Payment Method</span>
        <span class="info-value">{{.PaymentMethod}}</span>
    </div>
</div>

<p>No action is needed if you wish to continue your subscription.</p>

<p>If you wish to cancel or modify your subscription before renewal:</p>

<p style="text-align: center;">
    <a href="{{.PortalURL}}" class="btn">Manage Subscription</a>
</p>

<p style="font-size: 12px; color: #64748b;">
    By continuing your subscription, you authorize us to charge your payment method on the date shown above.
</p>
`

// TrialEndingTemplate reminds a customer that their free trial is about to end
// and the first charge is coming.
const TrialEndingTemplate = `
<h1>Your free trial is ending soon</h1>
<p>Hello {{.CustomerName}},</p>
<p>Your free trial of <strong>{{.PlanName}}</strong> ends on <strong>{{.TrialEndDate}}</strong>. To keep your subscription active, we'll charge your payment method after the trial ends.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Plan</span>
        <span class="info-value">{{.PlanName}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">First charge</span>
        <span class="info-value">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Trial ends</span>
        <span class="info-value">{{.TrialEndDate}}</span>
    </div>
</div>

<p>No action is needed if you wish to continue. To change or cancel before the trial ends:</p>

<p style="text-align: center;">
    <a href="{{.PortalURL}}" class="btn">Manage Subscription</a>
</p>
`

// DunningFirstReminderTemplate for first payment retry reminder (Day 1)
const DunningFirstReminderTemplate = `
<h1>Action Required: Payment Failed</h1>
<p>Hello {{.CustomerName}},</p>
<p>We attempted to charge your payment method but it was unsuccessful.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Amount Due</span>
        <span class="info-value amount">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Invoice</span>
        <span class="info-value">{{.InvoiceNumber}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Next Retry</span>
        <span class="info-value">{{.NextRetryDate}}</span>
    </div>
</div>

<p><strong>What to do:</strong></p>
<ul>
    <li>Update your payment method if your card has expired</li>
    <li>Ensure sufficient funds are available</li>
    <li>Contact your bank if payments are being blocked</li>
</ul>

<p style="text-align: center;">
    <a href="{{.UpdatePaymentURL}}" class="btn">Update Payment Method</a>
</p>

<p>We'll automatically retry the payment on {{.NextRetryDate}}.</p>
`

// DunningSecondReminderTemplate for second payment retry reminder (Day 3)
const DunningSecondReminderTemplate = `
<h1>⚠️ Payment Still Pending</h1>
<p>Hello {{.CustomerName}},</p>
<p>We've tried to process your payment <strong>{{.RetryCount}} times</strong> without success.</p>

<div class="info-box" style="border-left: 4px solid #f59e0b;">
    <div class="info-row">
        <span class="info-label">Amount Due</span>
        <span class="info-value amount">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Days Overdue</span>
        <span class="info-value" style="color: #f59e0b;">{{.DaysOverdue}} days</span>
    </div>
    <div class="info-row">
        <span class="info-label">Service Status</span>
        <span class="status status-warning">At Risk</span>
    </div>
</div>

<p><strong>Your access may be suspended</strong> if payment is not received by {{.SuspensionDate}}.</p>

<p style="text-align: center;">
    <a href="{{.PayNowURL}}" class="btn" style="background: #f59e0b;">Pay Now</a>
</p>

<p>Need help? Reply to this email or contact our support team.</p>
`

// DunningFinalNoticeTemplate for final notice before service suspension (Day 7)
const DunningFinalNoticeTemplate = `
<h1>🚨 Final Notice: Service Suspension</h1>
<p>Hello {{.CustomerName}},</p>
<p>Despite multiple attempts, we have been unable to collect payment for your account.</p>

<div class="info-box" style="border-left: 4px solid #ef4444;">
    <div class="info-row">
        <span class="info-label">Amount Due</span>
        <span class="info-value amount">{{.Amount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Days Overdue</span>
        <span class="info-value" style="color: #ef4444;">{{.DaysOverdue}} days</span>
    </div>
    <div class="info-row">
        <span class="info-label">Suspension Date</span>
        <span class="info-value" style="color: #ef4444;">{{.SuspensionDate}}</span>
    </div>
</div>

<p><strong>Your service will be suspended on {{.SuspensionDate}}</strong> unless payment is received.</p>

<p>After suspension:</p>
<ul>
    <li>You will lose access to your account</li>
    <li>Data may be deleted after 30 days</li>
    <li>You can reactivate anytime by paying the outstanding amount</li>
</ul>

<p style="text-align: center;">
    <a href="{{.PayNowURL}}" class="btn" style="background: #ef4444;">Pay Now to Avoid Suspension</a>
</p>
`

// CardExpiringTemplate for card expiry warning notification
const CardExpiringTemplate = `
<h1>Your Card Is Expiring Soon</h1>
<p>Hello {{.CustomerName}},</p>
<p>The payment method on your account will expire soon. Please update it to avoid any interruption to your service.</p>

<div class="info-box" style="border-left: 4px solid #f59e0b;">
    <div class="info-row">
        <span class="info-label">Card</span>
        <span class="info-value">{{.CardBrand}} ending in {{.CardLast4}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Expires</span>
        <span class="info-value" style="color: #f59e0b;">{{.ExpiryDate}}</span>
    </div>
</div>

<p>To ensure uninterrupted service, please update your payment method before your card expires.</p>

<p style="text-align: center;">
    <a href="{{.UpdatePaymentURL}}" class="btn" style="background: #f59e0b;">Update Payment Method</a>
</p>

<p style="font-size: 12px; color: #64748b;">
    If you've already updated your card, you can safely ignore this email.
</p>
`

// SubscriptionCancelledTemplate for cancellation confirmation
const SubscriptionCancelledTemplate = `
<h1>Subscription Cancelled</h1>
<p>Hello {{.CustomerName}},</p>
<p>Your subscription has been cancelled as requested.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Plan</span>
        <span class="info-value">{{.PlanName}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Access Until</span>
        <span class="info-value">{{.AccessUntil}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Status</span>
        <span class="status status-warning">Cancelled</span>
    </div>
</div>

<p>You will continue to have access until <strong>{{.AccessUntil}}</strong>, the end of your current billing period.</p>

<p>Changed your mind? You can reactivate anytime:</p>

<p style="text-align: center;">
    <a href="{{.ReactivateURL}}" class="btn">Reactivate Subscription</a>
</p>

<p style="font-size: 12px; color: #64748b;">
    We'd love to hear why you cancelled. Reply to this email with any feedback.
</p>
`

// GiftPurchasedTemplate notifies the recipient that they've received a gift subscription
const GiftPurchasedTemplate = `
<h1>You've Received a Gift!</h1>
<p>Hello,</p>
<p>Someone has gifted you a subscription. Use the code below to activate your gift.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Plan</span>
        <span class="info-value">{{.PlanName}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Duration</span>
        <span class="info-value">{{.Duration}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Gift Code</span>
        <span class="info-value" style="font-size: 18px; letter-spacing: 2px;">{{.GiftCode}}</span>
    </div>
</div>

<p style="text-align: center;">
    <a href="{{.RedeemURL}}" class="btn">Redeem Gift</a>
</p>

<p style="font-size: 14px; color: #64748b;">This gift code does not expire. You can redeem it anytime.</p>
`

// GSTInvoiceTemplate for GST-compliant invoice email
const GSTInvoiceTemplate = `
<h1>Tax Invoice</h1>
<p>Hello {{.CustomerName}},</p>
<p>Please find your GST-compliant tax invoice below.</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Invoice Number</span>
        <span class="info-value">{{.InvoiceNumber}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Invoice Date</span>
        <span class="info-value">{{.InvoiceDate}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">SAC Code</span>
        <span class="info-value">{{.SACCode}}</span>
    </div>
    {{if .SellerGSTIN}}
    <div class="info-row">
        <span class="info-label">Seller GSTIN</span>
        <span class="info-value">{{.SellerGSTIN}}</span>
    </div>
    {{end}}
    {{if .BuyerGSTIN}}
    <div class="info-row">
        <span class="info-label">Buyer GSTIN</span>
        <span class="info-value">{{.BuyerGSTIN}}</span>
    </div>
    {{end}}
</div>

<h2>Amount Details</h2>
<div class="info-box">
    <div class="info-row">
        <span class="info-label">Subtotal</span>
        <span class="info-value">{{.Subtotal}}</span>
    </div>
    {{if .CGSTAmount}}
    <div class="info-row">
        <span class="info-label">CGST @ {{.CGSTRate}}%</span>
        <span class="info-value">{{.CGSTAmount}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">SGST @ {{.SGSTRate}}%</span>
        <span class="info-value">{{.SGSTAmount}}</span>
    </div>
    {{end}}
    {{if .IGSTAmount}}
    <div class="info-row">
        <span class="info-label">IGST @ {{.IGSTRate}}%</span>
        <span class="info-value">{{.IGSTAmount}}</span>
    </div>
    {{end}}
    <div class="info-row" style="border-top: 2px solid #0f172a; margin-top: 8px; padding-top: 12px;">
        <span class="info-label" style="font-weight: 700;">Total</span>
        <span class="info-value amount">{{.Total}}</span>
    </div>
</div>

<p style="text-align: center;">
    <a href="{{.DownloadURL}}" class="btn">Download PDF Invoice</a>
</p>

<p style="font-size: 12px; color: #64748b;">
    Place of Supply: {{.PlaceOfSupply}}<br>
    {{if .IRN}}IRN: {{.IRN}}{{end}}
</p>
`

// DunningCampaignEmailTemplate for configurable dunning campaign emails
const DunningCampaignEmailTemplate = `
<h1>{{.Subject}}</h1>
<p>Hello {{.CustomerName}},</p>
<p>{{.Body}}</p>

<div class="info-box">
    <div class="info-row">
        <span class="info-label">Invoice</span>
        <span class="info-value">{{.InvoiceNumber}}</span>
    </div>
    <div class="info-row">
        <span class="info-label">Amount Due</span>
        <span class="info-value amount">{{.Amount}}</span>
    </div>
</div>

<p>Please take action to avoid service interruption.</p>
`
