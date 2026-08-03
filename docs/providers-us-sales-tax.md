# US sales-tax providers

Recurso computes US sales tax through a pluggable provider. The port is
`tax.SalesTaxProvider` (`internal/core/service/tax/provider.go`) and it is
deliberately narrow: it answers *"what is the tax on this amount shipped from A
to B"*, and nothing more. Nexus tracking, liability reporting, and exemption
records stay in Recurso.

Three providers implement it. None is a default or a recommendation. Pick on
the tradeoffs below. With no provider configured the US engine stays an honest
0% stub and invoices are marked `sales_tax_stub`.

| | TaxJar | Avalara AvaTax | Ziptax |
| -- | -- | -- | -- |
| Env key | `TAXJAR_API_KEY` | `AVALARA_ACCOUNT_ID` | `ZIPTAX_API_KEY` |
| Base URL override | `TAXJAR_API_URL` | `AVALARA_API_URL` | `ZIPTAX_API_URL` |
| Account required | Paid | Paid | Free tier available |
| Returns | Computed tax amount | Computed tax amount | Rate; Recurso computes the amount |
| Exemptions | `exemption_type` sent | `entityUseCode` + `exemptionNo` sent | Not supported (see below) |
| Status | Stable | Experimental | New |

All three are third-party SaaS egress and are hard-disabled under
`RESIDENCY_MODE=self_hosted`, at their construction site in `cmd/api/main.go`.
All three are wrapped in `CachedSalesTaxProvider` (24h TTL, keyed on origin +
destination + declared nexus states).

## Choosing

- **You already pay for one of them.** Use it. There is no accuracy or feature
  reason to switch.
- **You need filing, returns, or provider-held exemption certificates.** Ziptax
  does not do these through this adapter. TaxJar and Avalara have products that
  do, though Recurso does not currently drive them.
- **You are evaluating Recurso, or self-hosting at low volume.** Ziptax has a
  free tier, so US tax computes on day one without a commercial contract. Read
  the limits below before depending on it.

## Ziptax

`internal/adapter/taxprovider/ziptax.go`. Calls `GET /request/v60`, authenticated
with an `X-API-KEY` header (never a `key=` query parameter, which would put the
credential in access and proxy logs).

Create a key at [platform.zip.tax](https://platform.zip.tax/); the vendor's
walkthrough is [How to create an API key](https://docs.zip.tax/guides/tutorials/how-to-create-an-api-key).

### Limits

Ziptax's published pricing at the time of writing: the free **Starter** plan is
**100 API calls per month** with a **10 requests/minute** rate limit and no
overage. Paid tiers start at $29/month. Verify against
[zip.tax/pricing](https://www.zip.tax/pricing) before relying on these. They are
the vendor's numbers, not Recurso's, and they can change.

100 calls a month is an evaluation allowance. It is comfortable for standing
Recurso up and confirming US tax works; it is not enough to run a business on. A
deployment invoicing into more than a handful of ZIPs monthly will need a paid
tier.

The 24h rate cache is what makes the free tier usable at all, not merely an
optimisation. A tenant invoicing into 20 ZIP codes costs roughly 20 calls a month
with the cache and would exhaust the entire monthly allowance in a day without
it.

### Accuracy: ZIP level, not rooftop

`SalesTaxQuery` carries no street address: the destination is
`ToCountry`/`ToState`/`ToZip`. So this adapter performs a **postal-code lookup**,
which Ziptax documents as its least precise mode: the result is not adjusted for
unincorporated areas or special tax jurisdictions the way an address or
latitude/longitude lookup is.

This matters because a single ZIP can span several tax jurisdictions. If you need
rooftop accuracy, the query struct needs a street-address field; that is a change
to the port, and it would benefit the TaxJar and Avalara adapters equally. It is
tracked separately rather than worked around inside one provider.

Note also that Ziptax's street-address lookup requires the geocoding entitlement
on the account, which the free tier does not include. The postal-code path this
adapter uses needs no entitlement beyond an active key.

### Exemptions

The `/request/v60` rates endpoint takes no exemption parameter. When
`q.IsExempt()` is true the adapter short-circuits to zero tax **without making an
API call**. The answer is known, and on a 100-call allowance a wasted call is
expensive.

The honest gap: unlike TaxJar and Avalara, **a Ziptax lookup does not record the
exempt sale anywhere.** Recurso holds the exemption record itself, so nothing is
lost functionally, but there is no provider-side audit artifact. If you expect
your tax provider to hold exemption evidence for you, this adapter will not do
that.

Ziptax does offer exemption-certificate management, but only under its Merchant
Compliance product, which needs a connected-merchant ID and is not reachable with
a standard API key. It is deliberately not wired in here.

### Nexus

Ziptax is never asked about nexus, and forwards nothing from `q.NexusStates`.

Nexus is a registration fact: it depends on where the seller is registered and
which thresholds they have crossed, and it is not derivable from an address plus
an amount. A rate API can say what a jurisdiction charges; it cannot say whether
you are obliged to collect it. Conflating the two fails badly in both
directions: collecting where you are not registered means holding money without
authority, and not collecting where you are is a liability that surfaces at
audit.

Recurso already owns that determination, so `HasNexus` is derived from the
query's own `NexusStates`: true when the destination state is among them, false
when it is not, and true when the list is empty (the adapter has no grounds to
contradict Recurso). It is never hardcoded, and it is never inferred from the
rate being non-zero.

### Sourcing

Nothing from `FromCountry`/`FromState`/`FromZip` is sent: the v60 rates endpoint
has no origin parameter. Ziptax resolves origin-versus-destination sourcing
server-side and reports which rule it applied in the response's `sourcingRules`.
This differs from the TaxJar adapter, which does pass `from_*` through.

### Product taxability

Not sent. Mapping a product to a taxability code stays with the caller. Ziptax
supports product rules on paid tiers via a `taxabilityCode` parameter, but the
Recurso port has no product dimension to map from, so wiring it would mean
inventing one.

### Errors

The adapter branches on the **application code** in `metadata.response.code`, not
the HTTP status: Ziptax returns meaningful codes there and documents the numeric
codes as the stable contract. The HTTP status is only consulted when no code can
be read, so a proxy error page in front of the API still maps sensibly.

| Ziptax code | Sentinel | Retried |
| -- | -- | -- |
| 100 success | n/a | n/a |
| 101 invalid key | `ErrZiptaxAuth` | No |
| 102-105, 109, 110, 111 | `ErrZiptaxBadRequest` | No |
| 106 API error | `ErrZiptaxUnavailable` | Once |
| 107, 112, 113 not entitled | `ErrZiptaxNotEntitled` | No |
| 108 rate limit | `ErrZiptaxUnavailable` | Once |
| transport failure | `ErrZiptaxUnavailable` | Once |

`ErrZiptaxNotEntitled` is a fourth sentinel where TaxJar and Avalara have three.
Plan-entitlement failures are not caller mistakes, and reporting them as "invalid
request" gives an operator the one message that cannot lead them to a fix.

Retries are capped at one, with no backoff, matching the convention the other two
adapters set. Ziptax's docs recommend exponential backoff on 108, but Ziptax
sends no `Retry-After`, and an adapter that sleeps inside a checkout path is
worse than one that returns a typed error promptly. Degradation policy belongs to
the caller.

## Cache staleness

This applies to all three providers, and is worth knowing before you rely on any
of them.

`CachedSalesTaxProvider` has a **time-based** TTL, not an effective-date-based
one. Sales-tax rates change on effective dates, so a rate change can be served
stale for up to 24 hours after it takes effect, producing an order that
disagrees with the return eventually filed on it.

In practice rate changes are published and effective-dated weeks ahead, so the
exposure is a 24h window at each rate-change boundary rather than a persistent
drift. It is documented here rather than quietly relied upon. Keying the cache on
effective date instead would remove the window; that is a change to the shared
cache, not to any one provider.
