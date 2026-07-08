// Competitor comparison data for /vs/<slug> pages.
//
// Every row states a RECURSO capability (the `recurso` column) alongside the
// competitor's column. Competitor cells are only marked true / false / 'partial'
// where the claim is publicly checkable and defensible; uncertain rows are
// omitted entirely rather than guessed. These mirror the vetted rows already
// used in src/components/Comparison.jsx.

export const competitors = {
    lago: {
        slug: 'lago',
        name: 'Lago',
        title: 'Recurso vs Lago — MIT-licensed open-source billing',
        description:
            'Recurso and Lago are both open-source, self-hostable billing engines. Compare licensing (MIT vs AGPLv3), multi-gateway payments, India GST & e-invoicing, and a double-entry ledger.',
        headline: 'Recurso vs Lago',
        lede:
            'Both are open-source billing engines you can self-host for zero software fees. The differences that matter are the license (MIT vs AGPLv3), the breadth of the billing surface, and India-native compliance. Lago is strong at usage metering; Recurso is broader and India-first.',
        note: 'MIT licensed · Both self-hostable · Stripe + Razorpay + UPI AutoPay',
        bullets: [
            {
                title: 'MIT vs AGPLv3',
                body: 'Recurso is MIT licensed — permissive enough to fork, embed, or ship inside closed-source products. Lago is AGPLv3, a network copyleft license that many corporate legal teams restrict.',
            },
            {
                title: 'India-native compliance',
                body: 'Recurso builds in GST and e-invoicing for Indian businesses. Lago does not list India tax compliance as a focus.',
            },
            {
                title: 'Multi-gateway payments',
                body: 'Recurso ships Stripe, Razorpay, and UPI AutoPay out of the box for both global and Indian collections.',
            },
            {
                title: 'Double-entry ledger',
                body: 'Every invoice and payment in Recurso posts to a double-entry ledger (TigerBeetle-backed). Lago is strongest at high-volume usage metering.',
            },
        ],
        rows: [
            { name: 'Self-hosted option', recurso: true, competitor: true },
            { name: 'MIT-licensed', recurso: true, competitor: false },
            { name: 'No platform fees', recurso: true, competitor: true },
            { name: 'Usage-based billing', recurso: true, competitor: true },
            { name: 'Full subscription lifecycle', recurso: true, competitor: true },
            { name: 'Multi-gateway (Stripe + Razorpay)', recurso: true, competitor: 'partial' },
            { name: 'UPI AutoPay', recurso: true, competitor: false },
            { name: 'India GST & e-invoicing', recurso: true, competitor: false },
            { name: 'Double-entry ledger', recurso: true, competitor: false },
        ],
    },

    flexprice: {
        slug: 'flexprice',
        name: 'FlexPrice',
        title: 'Recurso vs FlexPrice — open-source billing compared',
        description:
            'Recurso and FlexPrice are open-source billing engines. Recurso adds a broader subscription lifecycle, native India GST & e-invoicing, and multi-gateway payments.',
        headline: 'Recurso vs FlexPrice',
        lede:
            'Both are open-source, self-hostable billing engines. FlexPrice focuses on usage-based billing; Recurso covers the fuller subscription lifecycle — dunning, quotes, credit notes, entitlements — with India-native compliance and multi-gateway payments.',
        note: 'Open source · Self-hostable · Broader lifecycle · India-first',
        bullets: [
            {
                title: 'Broader subscription lifecycle',
                body: 'Recurso covers dunning, quotes, credit notes, and entitlements alongside subscriptions and usage — not usage metering alone.',
            },
            {
                title: 'India-native compliance',
                body: 'GST and e-invoicing are built into Recurso for Indian businesses.',
            },
            {
                title: 'Multi-gateway payments',
                body: 'Stripe, Razorpay, and UPI AutoPay ship in the box for global and Indian collections.',
            },
        ],
        rows: [
            { name: 'Self-hosted option', recurso: true, competitor: true },
            { name: 'No platform fees', recurso: true, competitor: true },
            { name: 'Usage-based billing', recurso: true, competitor: true },
            { name: 'Full subscription lifecycle', recurso: true, competitor: 'partial' },
            { name: 'India GST & e-invoicing', recurso: true, competitor: false },
        ],
    },

    killbill: {
        slug: 'killbill',
        name: 'Kill Bill',
        title: 'Recurso vs Kill Bill — modern Go billing vs Java',
        description:
            'Recurso and Kill Bill are both open-source and permissively licensed. Compare a modern Go stack with one-command deploy against Kill Bill’s mature Java platform, plus native India GST & e-invoicing.',
        headline: 'Recurso vs Kill Bill',
        lede:
            'Kill Bill is a mature, battle-tested, Apache-2.0 billing platform with years of production mileage. Recurso is a modern Go stack with a one-command deploy and India-native compliance. Both are open source and permissively licensed.',
        note: 'Apache-2.0 (Kill Bill) · MIT (Recurso) · both open source',
        bullets: [
            {
                title: 'Modern Go stack, one-command deploy',
                body: 'Recurso is a single Go service you can stand up with one command, versus Kill Bill’s heavier Java / OSGi plugin platform.',
            },
            {
                title: 'India-native compliance',
                body: 'GST and e-invoicing are built into Recurso. Kill Bill leaves tax and invoicing localization to plugins and integrators.',
            },
            {
                title: 'Managed Cloud option',
                body: 'Recurso offers a managed Cloud option alongside self-hosting (waitlist), so you can start hosted and move to self-host, or vice versa.',
            },
            {
                title: 'Kill Bill’s maturity is real',
                body: 'Kill Bill has a large community and long production track record. If you need proven mileage today, that is a genuine strength.',
            },
        ],
        rows: [
            { name: 'Self-hosted option', recurso: true, competitor: true },
            { name: 'No platform fees', recurso: true, competitor: true },
            { name: 'Usage-based billing', recurso: true, competitor: true },
            { name: 'Full subscription lifecycle', recurso: true, competitor: true },
            { name: 'One-command deploy', recurso: true, competitor: false },
            { name: 'Multi-gateway (Stripe + Razorpay)', recurso: true, competitor: 'partial' },
            { name: 'UPI AutoPay', recurso: true, competitor: false },
            { name: 'India GST & e-invoicing', recurso: true, competitor: false },
        ],
    },

    chargebee: {
        slug: 'chargebee',
        name: 'Chargebee',
        title: 'Recurso vs Chargebee — open-source, self-hosted billing',
        description:
            'Recurso is an open-source, self-hostable alternative to Chargebee: no platform fees, own your data, with native India GST & e-invoicing.',
        headline: 'Recurso vs Chargebee',
        lede:
            'Chargebee is a proprietary billing SaaS with no self-host option and revenue-based platform pricing. Recurso is open source and self-hostable — no platform fees, you own your data, and India GST & e-invoicing are native.',
        note: 'Open source · Self-hostable · No platform fees · Own your data',
        bullets: [
            {
                title: 'Open source and self-hostable',
                body: 'Run Recurso on your own infrastructure under the MIT license. Chargebee is a proprietary hosted SaaS with no self-host option.',
            },
            {
                title: 'No platform fees',
                body: 'Recurso does not charge a percentage of your revenue. Chargebee’s pricing scales with your billing volume.',
            },
            {
                title: 'Own your data',
                body: 'Your billing data and ledger live in your own database, not a third-party platform.',
            },
            {
                title: 'India-native compliance',
                body: 'GST and e-invoicing are built into Recurso.',
            },
        ],
        rows: [
            { name: 'Self-hosted option', recurso: true, competitor: false },
            { name: 'MIT-licensed', recurso: true, competitor: false },
            { name: 'No platform fees', recurso: true, competitor: false },
            { name: 'Usage-based billing', recurso: true, competitor: true },
            { name: 'Full subscription lifecycle', recurso: true, competitor: true },
            { name: 'Multi-gateway (Stripe + Razorpay)', recurso: true, competitor: true },
            { name: 'UPI AutoPay', recurso: true, competitor: 'partial' },
            { name: 'India GST & e-invoicing', recurso: true, competitor: true },
            { name: 'Double-entry ledger', recurso: true, competitor: false },
        ],
    },

    'stripe-billing': {
        slug: 'stripe-billing',
        name: 'Stripe Billing',
        title: 'Recurso vs Stripe Billing — multi-gateway, self-hosted',
        description:
            'Recurso is a multi-gateway, self-hostable, open-source alternative to Stripe Billing — not locked to a single processor, India-first, with native GST & e-invoicing.',
        headline: 'Recurso vs Stripe Billing',
        lede:
            'Stripe Billing is proprietary and tied to Stripe as the payment processor, with platform fees and no self-host option. Recurso is open source, self-hostable, and multi-gateway — it integrates Stripe as one of its gateways, so you are never locked to a single processor.',
        note: 'Multi-gateway · Self-hostable · Open source · Integrates Stripe',
        bullets: [
            {
                title: 'Multi-gateway, not single-processor',
                body: 'Recurso routes across Stripe, Razorpay, and UPI AutoPay. Stripe Billing ties you to Stripe as the processor. Recurso integrates Stripe too — you just are not locked to it.',
            },
            {
                title: 'Open source and self-hostable',
                body: 'Run Recurso on your own infrastructure under the MIT license. Stripe Billing is a proprietary hosted product.',
            },
            {
                title: 'India-first',
                body: 'GST and e-invoicing, plus Razorpay and UPI AutoPay, are native to Recurso.',
            },
        ],
        rows: [
            { name: 'Self-hosted option', recurso: true, competitor: false },
            { name: 'MIT-licensed', recurso: true, competitor: false },
            { name: 'No platform fees', recurso: true, competitor: false },
            { name: 'Usage-based billing', recurso: true, competitor: true },
            { name: 'Full subscription lifecycle', recurso: true, competitor: true },
            { name: 'Multi-gateway (Stripe + Razorpay)', recurso: true, competitor: false },
            { name: 'UPI AutoPay', recurso: true, competitor: false },
            { name: 'India GST & e-invoicing', recurso: true, competitor: 'partial' },
            { name: 'Double-entry ledger', recurso: true, competitor: false },
        ],
    },
}

export const competitorList = Object.values(competitors)
