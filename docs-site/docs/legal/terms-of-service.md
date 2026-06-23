---
sidebar_position: 1
title: Terms of Service
---

# Terms of Service

**Effective Date:** June 23, 2026

**Last Updated:** June 23, 2026

:::note Disclaimer
This document is provided for informational purposes and constitutes a binding legal agreement between you and Recurso. However, you are encouraged to consult your own legal counsel before accepting these terms, particularly if you are using Recurso to process billing or payment data on behalf of your end users.
:::

---

## 1. Acceptance of Terms

### 1.1 Agreement to Terms

By accessing or using the Recurso platform ("Service"), available at [recurso.dev](https://recurso.dev), including the API, dashboard, documentation, and any related services, you ("Customer," "you," or "your") agree to be bound by these Terms of Service ("Terms"). If you are accepting these Terms on behalf of a company, organization, or other legal entity, you represent and warrant that you have the authority to bind that entity to these Terms, in which case "you" and "your" shall refer to that entity.

### 1.2 Eligibility

You must be at least eighteen (18) years of age and capable of forming a binding contract under applicable law to use the Service. By using the Service, you represent and warrant that you meet these requirements.

### 1.3 Additional Terms

Certain features of the Service may be subject to additional terms and conditions, including but not limited to Service Level Agreements ("SLAs"), Data Processing Agreements ("DPAs"), and Enterprise agreements. Such additional terms are incorporated by reference into these Terms.

---

## 2. Description of Service

### 2.1 Platform Overview

Recurso is a billing engine platform that enables businesses to manage subscriptions, invoicing, metering, payment processing, and revenue operations. The Service includes:

- **API Access:** RESTful APIs and SDKs for programmatic integration of billing functionality into your applications.
- **Dashboard:** A web-based management console for configuring billing plans, viewing analytics, managing customers, and administering your account.
- **Billing Engine:** Automated subscription lifecycle management, usage-based metering, invoice generation, proration calculations, and payment orchestration.
- **Integrations:** Connectors to third-party payment processors (such as Stripe and Razorpay), accounting platforms (such as QuickBooks and Xero), and other business tools.
- **Documentation:** Technical documentation, API references, guides, and tutorials.

### 2.2 Service Tiers

Recurso is offered under the following tiers:

| Feature | Open Source (Free) | Pro ($299/month) | Enterprise (Custom) |
|---|---|---|---|
| Core billing engine | Yes | Yes | Yes |
| API access | Yes | Yes | Yes |
| Dashboard | Yes | Yes | Yes |
| Community support | Yes | Yes | Yes |
| Priority support | No | Yes | Yes |
| Advanced analytics | No | Yes | Yes |
| Custom integrations | No | Limited | Yes |
| SLA guarantees | No | 99.9% uptime | Custom |
| Dedicated infrastructure | No | No | Yes |
| Custom contracts | No | No | Yes |

### 2.3 Open Source Components

The core billing engine of Recurso is released under the MIT License. The terms of the MIT License govern your use of the open-source components. Pro and Enterprise features, proprietary integrations, and the hosted platform are governed exclusively by these Terms.

---

## 3. Account Registration and Responsibilities

### 3.1 Account Creation

To access certain features of the Service, you must create an account by providing accurate, current, and complete information. You agree to update your account information promptly to keep it accurate and complete.

### 3.2 Account Security

You are responsible for:

- Maintaining the confidentiality of your account credentials, API keys, and access tokens.
- All activities that occur under your account, whether or not authorized by you.
- Notifying Recurso immediately at [security@recurso.dev](mailto:security@recurso.dev) upon becoming aware of any unauthorized use of your account or any other breach of security.

### 3.3 API Key Management

You must keep all API keys and authentication tokens secure. You shall not share API keys publicly, embed them in client-side code, or commit them to public repositories. Recurso reserves the right to revoke API keys that are found to be compromised or misused.

### 3.4 Account Restrictions

You may not:

- Create multiple free-tier accounts to circumvent usage limits.
- Share account credentials with unauthorized third parties.
- Use another person's or entity's account without permission.
- Sell, transfer, or sublicense your account access without Recurso's prior written consent.

---

## 4. Pricing and Payment Terms

### 4.1 Free Tier (Open Source)

The Open Source tier is provided free of charge, subject to the usage limits described in Section 5. Recurso reserves the right to modify free-tier limits with thirty (30) days' prior notice.

### 4.2 Pro Tier

The Pro tier is offered at **$299 per month** (USD), billed monthly or annually at the Customer's election. Annual billing is offered at a discounted rate as published on the Recurso pricing page. All fees are exclusive of applicable taxes, duties, and levies.

### 4.3 Enterprise Tier

Enterprise pricing is determined on a custom basis. Enterprise Customers will receive a separate order form or statement of work specifying pricing, payment terms, and any additional commitments.

### 4.4 Payment Processing

- Payments are processed through third-party payment processors. By providing payment information, you authorize Recurso and its payment processors to charge the applicable fees.
- All fees are due in advance for the applicable billing period unless otherwise specified in an Enterprise agreement.
- Fees are non-refundable except as expressly set forth in these Terms or as required by applicable law.

### 4.5 Late Payments and Suspension

- If payment is not received within seven (7) days of the due date, Recurso may send a written notice of overdue payment.
- If payment remains outstanding for more than fifteen (15) days after such notice, Recurso reserves the right to suspend access to paid features until the account is brought current.
- Recurso may charge interest on overdue amounts at the rate of 1.5% per month or the maximum rate permitted by applicable law, whichever is lower.

### 4.6 Taxes

You are responsible for all applicable taxes, including but not limited to GST, VAT, sales tax, and withholding tax. If Recurso is required to collect or remit taxes on your behalf, such taxes will be added to your invoice.

### 4.7 Price Changes

Recurso reserves the right to change its pricing with at least thirty (30) days' prior written notice. Price changes will take effect at the start of the next billing cycle following the notice period. If you do not agree to a price change, you may cancel your subscription before the new pricing takes effect.

---

## 5. Usage Limits and Fair Use

### 5.1 API Rate Limits

API access is subject to rate limits that vary by tier:

| Limit | Open Source | Pro | Enterprise |
|---|---|---|---|
| API requests per minute | 60 | 1,000 | Custom |
| API requests per month | 10,000 | 500,000 | Custom |
| Webhooks per month | 1,000 | 100,000 | Custom |
| Customers managed | 100 | Unlimited | Unlimited |
| Concurrent connections | 5 | 50 | Custom |

Recurso reserves the right to adjust these limits and will provide reasonable notice of any changes.

### 5.2 Fair Use Policy

All tiers are subject to a fair use policy. You agree not to:

- Use the Service in a manner that degrades performance for other customers.
- Attempt to circumvent rate limits through multiple accounts, distributed requests, or other means.
- Use the API for purposes unrelated to billing and subscription management.
- Generate excessive load through inefficient API usage patterns when reasonable alternatives exist.

### 5.3 Exceeding Limits

If you exceed the usage limits for your tier, Recurso will notify you and may:

- Temporarily throttle your API access.
- Require you to upgrade to a higher tier.
- Charge overage fees as published on the Recurso pricing page.

---

## 6. Data Handling and Processing

### 6.1 Customer Data

In the course of providing the Service, Recurso will process data that you or your end users submit to the Service ("Customer Data"). Customer Data includes, but is not limited to, billing information, subscription details, payment records, customer profiles, and usage data.

### 6.2 Data Processing Role

- **As Data Processor:** With respect to Customer Data (including your end users' billing and payment information), Recurso acts as a data processor on your behalf. You remain the data controller and are responsible for ensuring that you have a lawful basis for collecting and processing such data.
- **As Data Controller:** With respect to your account information, usage analytics, and data collected directly from you for the purpose of providing and improving the Service, Recurso acts as an independent data controller.

### 6.3 Data Processing Agreement

Customers on the Pro and Enterprise tiers may request a Data Processing Agreement ("DPA") that provides additional commitments regarding the processing of personal data. Enterprise Customers may negotiate custom DPA terms.

### 6.4 Data Security

Recurso implements industry-standard technical and organizational measures to protect Customer Data, including encryption in transit and at rest, access controls, audit logging, and regular security assessments. Further details are available in our [Privacy Policy](/legal/privacy-policy).

### 6.5 Data Location

Customer Data is processed and stored in data centers located in regions specified in your account settings or as otherwise agreed upon. Recurso will not materially change the location of Customer Data processing without prior notice.

### 6.6 Subprocessors

Recurso may engage third-party subprocessors to assist in providing the Service. A current list of subprocessors is available upon request. Recurso will notify Customers of any material changes to its subprocessor list.

---

## 7. Intellectual Property

### 7.1 Open Source License

The core billing engine source code is licensed under the **MIT License**. You may use, modify, and distribute the open-source components in accordance with the MIT License terms. The full text of the MIT License is available in the project repository.

### 7.2 Proprietary Components

All Pro and Enterprise features, the hosted platform, proprietary integrations, the dashboard beyond the open-source version, advanced analytics, and any other components not explicitly released under an open-source license are the exclusive property of Recurso and are protected by intellectual property laws. You are granted a limited, non-exclusive, non-transferable, revocable license to use these components solely in connection with your authorized use of the Service.

### 7.3 Customer Data Ownership

You retain all rights, title, and interest in and to your Customer Data. Recurso does not claim ownership of Customer Data and will not use Customer Data for any purpose other than providing and improving the Service, except in an aggregated and anonymized form.

### 7.4 Feedback

If you provide Recurso with feedback, suggestions, or ideas ("Feedback"), you grant Recurso a perpetual, irrevocable, worldwide, royalty-free license to use, modify, and incorporate such Feedback into the Service without restriction or obligation to you.

### 7.5 Trademarks

"Recurso," the Recurso logo, and other Recurso trademarks, service marks, and trade names are the property of Recurso. You may not use these marks without prior written consent, except as reasonably necessary to refer to the Service in a descriptive manner.

---

## 8. Prohibited Uses

You agree not to use the Service to:

1. **Violate any law or regulation**, including but not limited to laws governing financial services, payment processing, data protection, anti-money laundering, and sanctions.
2. **Process payments for illegal goods or services**, including but not limited to illegal drugs, weapons, counterfeit goods, or any items prohibited by applicable law.
3. **Engage in fraud**, including creating fake transactions, manipulating billing data, or misrepresenting charges to end users.
4. **Infringe intellectual property rights** of any third party.
5. **Transmit malware, viruses, or malicious code** through the API or any other means.
6. **Attempt to gain unauthorized access** to Recurso's systems, infrastructure, other customers' data, or any related networks.
7. **Reverse engineer, decompile, or disassemble** any proprietary component of the Service, except to the extent expressly permitted by applicable law.
8. **Resell or redistribute the Service** without Recurso's prior written consent, except for the open-source components as permitted under the MIT License.
9. **Use the Service to build a competing product** by systematically extracting data, functionality, or design elements from the proprietary components of the Service.
10. **Engage in abusive or harassing behavior** toward Recurso staff or other customers.

Recurso reserves the right to suspend or terminate your account immediately if you engage in any prohibited use.

---

## 9. Service Availability and SLA

### 9.1 Availability Target

Recurso strives to maintain high availability of the Service. The availability commitments by tier are:

- **Open Source (Self-Hosted):** No availability commitment from Recurso. Availability depends on your infrastructure.
- **Pro:** 99.9% monthly uptime for the hosted API and dashboard, excluding scheduled maintenance.
- **Enterprise:** Custom SLA as specified in your Enterprise agreement.

### 9.2 Uptime Calculation

Uptime is calculated as:

```
Uptime % = ((Total Minutes in Month - Downtime Minutes) / Total Minutes in Month) x 100
```

"Downtime" means a period during which the Service is materially unavailable, as measured by Recurso's monitoring systems. Scheduled maintenance windows, announced at least forty-eight (48) hours in advance, are excluded from downtime calculations.

### 9.3 Service Credits (Pro Tier)

If Recurso fails to meet the 99.9% uptime commitment for Pro Customers in any calendar month, the following service credits will apply:

| Monthly Uptime | Service Credit |
|---|---|
| 99.0% - 99.9% | 10% of monthly fee |
| 95.0% - 99.0% | 25% of monthly fee |
| Below 95.0% | 50% of monthly fee |

Service credits must be requested within thirty (30) days of the end of the affected month and will be applied to future invoices. Service credits are your sole and exclusive remedy for downtime.

### 9.4 Exclusions

The availability commitment does not apply to:

- Downtime caused by factors outside Recurso's reasonable control, including force majeure events, internet outages, or third-party service failures.
- Downtime resulting from your actions, including misconfiguration, excessive API usage, or security incidents originating from your systems.
- Scheduled maintenance performed during announced maintenance windows.
- Features labeled as "beta," "preview," or "experimental."

---

## 10. Limitation of Liability

### 10.1 Disclaimer of Warranties

TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW, THE SERVICE IS PROVIDED "AS IS" AND "AS AVAILABLE" WITHOUT WARRANTIES OF ANY KIND, WHETHER EXPRESS, IMPLIED, STATUTORY, OR OTHERWISE, INCLUDING BUT NOT LIMITED TO IMPLIED WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, TITLE, AND NON-INFRINGEMENT.

RECURSO DOES NOT WARRANT THAT THE SERVICE WILL BE UNINTERRUPTED, ERROR-FREE, SECURE, OR FREE OF VIRUSES OR OTHER HARMFUL COMPONENTS.

### 10.2 Limitation of Liability

TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW:

**(a)** IN NO EVENT SHALL RECURSO, ITS DIRECTORS, OFFICERS, EMPLOYEES, AGENTS, OR AFFILIATES BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES, INCLUDING BUT NOT LIMITED TO LOSS OF PROFITS, REVENUE, DATA, BUSINESS OPPORTUNITIES, OR GOODWILL, ARISING OUT OF OR IN CONNECTION WITH THESE TERMS OR THE USE OF THE SERVICE, REGARDLESS OF THE THEORY OF LIABILITY (CONTRACT, TORT, STRICT LIABILITY, OR OTHERWISE), EVEN IF RECURSO HAS BEEN ADVISED OF THE POSSIBILITY OF SUCH DAMAGES.

**(b)** RECURSO'S TOTAL AGGREGATE LIABILITY ARISING OUT OF OR IN CONNECTION WITH THESE TERMS OR THE SERVICE SHALL NOT EXCEED THE GREATER OF: (i) THE TOTAL FEES PAID BY YOU TO RECURSO IN THE TWELVE (12) MONTHS PRECEDING THE EVENT GIVING RISE TO LIABILITY, OR (ii) ONE HUNDRED US DOLLARS ($100).

### 10.3 Essential Purpose

THE LIMITATIONS IN THIS SECTION APPLY EVEN IF ANY LIMITED REMEDY FAILS OF ITS ESSENTIAL PURPOSE. SOME JURISDICTIONS DO NOT ALLOW THE EXCLUSION OR LIMITATION OF CERTAIN DAMAGES, SO SOME OF THE ABOVE LIMITATIONS MAY NOT APPLY TO YOU.

---

## 11. Indemnification

### 11.1 Your Indemnification Obligations

You agree to indemnify, defend, and hold harmless Recurso, its directors, officers, employees, agents, and affiliates from and against any and all claims, damages, losses, liabilities, costs, and expenses (including reasonable attorneys' fees) arising out of or relating to:

1. Your use of the Service in violation of these Terms.
2. Your violation of any applicable law or regulation.
3. Your Customer Data or the manner in which you collect, process, or handle data through the Service.
4. Any dispute between you and your end users relating to billing, charges, or subscriptions managed through the Service.
5. Your breach of any representation or warranty made under these Terms.

### 11.2 Indemnification Procedure

Recurso will: (a) promptly notify you of any claim subject to indemnification; (b) provide reasonable cooperation in the defense of such claim at your expense; and (c) grant you sole control of the defense and settlement of such claim, provided that you may not settle any claim that imposes obligations on Recurso without Recurso's prior written consent.

---

## 12. Termination

### 12.1 Termination by You

You may terminate your account at any time by:

- Cancelling your subscription through the dashboard.
- Sending a written notice to [legal@recurso.dev](mailto:legal@recurso.dev).

For paid tiers, termination will take effect at the end of the current billing period. No refunds will be issued for partial billing periods unless required by applicable law.

### 12.2 Termination by Recurso

Recurso may terminate or suspend your account:

- **For Cause:** Immediately upon written notice if you materially breach these Terms and fail to cure such breach within fifteen (15) days of receiving notice, or immediately if the breach is not capable of cure (such as prohibited uses under Section 8).
- **For Convenience:** With thirty (30) days' prior written notice for any reason or no reason, provided that Recurso will refund any prepaid fees for the unused portion of the then-current billing period.
- **Discontinuation:** If Recurso discontinues the Service, with at least ninety (90) days' prior notice and a pro-rata refund of prepaid fees.

### 12.3 Effect of Termination

Upon termination:

- Your right to access paid features of the Service will cease immediately (or at the end of the billing period, as applicable).
- Recurso will retain your Customer Data for a period of thirty (30) days following termination to allow for data export (see Section 13).
- After the thirty (30) day retention period, Recurso will delete your Customer Data in accordance with its data retention policies, unless retention is required by law.
- Sections that by their nature should survive termination will survive, including but not limited to Sections 7 (Intellectual Property), 10 (Limitation of Liability), 11 (Indemnification), 14 (Governing Law), and 15 (Dispute Resolution).

---

## 13. Data Portability and Export

### 13.1 Data Export

You may export your Customer Data at any time during the term of your subscription through:

- The dashboard's data export functionality.
- The API's data export endpoints.
- By submitting a request to [support@recurso.dev](mailto:support@recurso.dev).

### 13.2 Export Formats

Customer Data will be made available for export in standard, machine-readable formats, including JSON and CSV.

### 13.3 Post-Termination Export

Upon termination of your account, you will have thirty (30) days to export your Customer Data. Recurso will provide reasonable assistance with data export during this period. After the thirty (30) day period, Recurso will delete your Customer Data unless retention is required by applicable law.

### 13.4 No Vendor Lock-In

Recurso is committed to data portability. The open-source nature of the core billing engine ensures that you can migrate your billing logic and data to a self-hosted instance or alternative platform.

---

## 14. Governing Law

### 14.1 Applicable Law

These Terms shall be governed by and construed in accordance with the laws of India, without regard to its conflict of law provisions.

### 14.2 Jurisdiction

Subject to the dispute resolution provisions in Section 15, the courts of Bengaluru, Karnataka, India shall have exclusive jurisdiction over any legal proceedings arising out of or in connection with these Terms.

### 14.3 Compliance with Local Laws

You are responsible for ensuring that your use of the Service complies with all applicable local, state, national, and international laws and regulations, including but not limited to the Information Technology Act, 2000 (India), the Digital Personal Data Protection Act, 2023 (India), the General Data Protection Regulation (EU), and the California Consumer Privacy Act (US), as applicable.

---

## 15. Dispute Resolution

### 15.1 Informal Resolution

Before initiating formal proceedings, you agree to first attempt to resolve any dispute informally by contacting Recurso at [legal@recurso.dev](mailto:legal@recurso.dev). Recurso will attempt to resolve the dispute informally within thirty (30) days of receiving your notice.

### 15.2 Arbitration

If the dispute cannot be resolved informally, either party may submit the dispute to binding arbitration under the Arbitration and Conciliation Act, 1996 (India). The arbitration shall be:

- Conducted by a sole arbitrator mutually agreed upon by the parties, or appointed in accordance with the Act if the parties cannot agree.
- Held in Bengaluru, Karnataka, India.
- Conducted in the English language.
- Governed by the substantive laws of India.

### 15.3 Exceptions

The following disputes are exempt from the arbitration requirement:

- Actions seeking injunctive or equitable relief to protect intellectual property rights.
- Claims that fall within the jurisdiction of small claims courts.
- Disputes required by applicable law to be resolved in a specific forum.

### 15.4 Class Action Waiver

To the maximum extent permitted by applicable law, you agree that any dispute resolution proceedings will be conducted on an individual basis and not as part of a class, consolidated, or representative action.

---

## 16. Changes to Terms

### 16.1 Right to Modify

Recurso reserves the right to modify these Terms at any time. Material changes will be communicated through:

- Email notification to the address associated with your account.
- A prominent notice on the Recurso website or dashboard.
- An update to the "Last Updated" date at the top of these Terms.

### 16.2 Acceptance of Changes

For material changes, Recurso will provide at least thirty (30) days' notice before the changes take effect. Your continued use of the Service after the effective date of the changes constitutes acceptance of the modified Terms. If you do not agree to the modified Terms, you must discontinue use of the Service and terminate your account before the changes take effect.

### 16.3 Non-Material Changes

Non-material changes (such as typographical corrections, formatting, or clarifications that do not alter the substance of the Terms) may be made without prior notice.

---

## 17. General Provisions

### 17.1 Entire Agreement

These Terms, together with the Privacy Policy, any applicable DPA, and any Enterprise agreement, constitute the entire agreement between you and Recurso with respect to the Service and supersede all prior agreements, understandings, and communications, whether written or oral.

### 17.2 Severability

If any provision of these Terms is held to be invalid, illegal, or unenforceable, the remaining provisions shall continue in full force and effect. The invalid provision shall be modified to the minimum extent necessary to make it valid and enforceable while preserving the original intent.

### 17.3 Waiver

The failure of Recurso to enforce any right or provision of these Terms shall not constitute a waiver of such right or provision. A waiver of any provision shall be effective only if made in writing and signed by Recurso.

### 17.4 Assignment

You may not assign or transfer your rights or obligations under these Terms without Recurso's prior written consent. Recurso may assign its rights and obligations under these Terms without your consent in connection with a merger, acquisition, reorganization, or sale of all or substantially all of its assets.

### 17.5 Force Majeure

Recurso shall not be liable for any failure or delay in performance resulting from causes beyond its reasonable control, including but not limited to acts of God, natural disasters, pandemics, war, terrorism, riots, government actions, internet outages, power failures, or third-party service failures.

### 17.6 Notices

All notices under these Terms shall be in writing and sent to:

- **To Recurso:** [legal@recurso.dev](mailto:legal@recurso.dev)
- **To You:** The email address associated with your account.

Notices shall be deemed received upon delivery if sent by email, provided that the sender does not receive a non-delivery notification.

### 17.7 Independent Contractors

The relationship between you and Recurso is that of independent contractors. Nothing in these Terms creates a partnership, joint venture, employment, or agency relationship.

---

## 18. Contact Information

If you have questions about these Terms of Service, please contact us:

- **Email:** [legal@recurso.dev](mailto:legal@recurso.dev)
- **Website:** [https://recurso.dev](https://recurso.dev)
- **Mailing Address:** Recurso, Bengaluru, Karnataka, India

---

*These Terms of Service are effective as of June 23, 2026.*
