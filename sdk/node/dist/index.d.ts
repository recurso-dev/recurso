/** Common list-endpoint query parameters (all optional, server-side). */
export interface ListParams {
    page?: number;
    limit?: number;
    q?: string;
    status?: string;
    [key: string]: unknown;
}
/**
 * Official Node.js SDK for the Recurso billing API.
 *
 * Method coverage mirrors the REST surface: list endpoints accept filter
 * params, mutations are grouped per resource, and lifecycle actions
 * (cancel, pause, resume, ...) live on their resource.
 */
export declare class Recurso {
    private client;
    constructor(apiKey: string, baseURL?: string);
    private get;
    private post;
    private put;
    private del;
    account: {
        get: () => Promise<any>;
        update: (data: Record<string, unknown>) => Promise<any>;
    };
    customers: {
        create: (data: {
            email: string;
            name: string;
            line1?: string;
            city?: string;
            state?: string;
            zip?: string;
            country?: string;
            [key: string]: unknown;
        }) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
        updatePaymentMethod: (id: string, data: Record<string, unknown>) => Promise<any>;
        churn: (id: string) => Promise<any>;
        consents: (id: string) => Promise<any>;
    };
    plans: {
        create: (data: {
            name: string;
            code: string;
            amount: number;
            currency: string;
            interval_unit: "day" | "week" | "month" | "year";
            interval_count?: number;
            [key: string]: unknown;
        }) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
    };
    subscriptions: {
        create: (data: {
            customer_id: string;
            plan_id: string;
            coupon_code?: string;
            start_date?: string;
            payment_terms?: string;
            [key: string]: unknown;
        }) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
        update: (id: string, data: Record<string, unknown>) => Promise<any>;
        cancel: (id: string, data?: Record<string, unknown>) => Promise<any>;
        pause: (id: string, data?: Record<string, unknown>) => Promise<any>;
        resume: (id: string) => Promise<any>;
        reactivate: (id: string) => Promise<any>;
        /** Bill N future periods immediately (advance invoicing). */
        advance: (id: string, data: {
            periods: number;
        }) => Promise<any>;
        charges: (id: string) => Promise<any>;
        addCharge: (id: string, data: Record<string, unknown>) => Promise<any>;
        /**
         * Current billing period's usage per dimension plus lifetime
         * totals, with the customer's entitlement limit/remaining joined
         * in where a feature_key matches the dimension name.
         */
        usage: (id: string) => Promise<any>;
    };
    invoices: {
        list: (params?: ListParams) => Promise<any>;
        /** Public PDF download URL for an invoice. */
        pdfUrl: (id: string) => string;
        eInvoiceStatus: (id: string) => Promise<any>;
        retryEInvoice: (id: string) => Promise<any>;
        cancelEInvoice: (id: string, data?: Record<string, unknown>) => Promise<any>;
    };
    coupons: {
        create: (data: {
            code: string;
            discount_type: "percent" | "amount";
            discount_value: number;
            duration: "forever" | "once";
            [key: string]: unknown;
        }) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
    };
    usage: {
        /** Record a metered usage event against a subscription. */
        record: (data: {
            subscription_id: string;
            customer_id: string;
            dimension: string;
            quantity: number;
        }) => Promise<any>;
        /**
         * Time-windowed usage buckets: {data: [{period, dimension,
         * quantity}], from, to, granularity}. At least one of
         * subscription_id or customer_id is required; the window defaults
         * to the last 30 days at day granularity.
         */
        query: (params: {
            subscription_id?: string;
            customer_id?: string;
            dimension?: string;
            from?: string;
            to?: string;
            granularity?: "day" | "month";
        }) => Promise<any>;
        /** The tenant's dimension catalog with first/last seen and event counts. */
        dimensions: () => Promise<any>;
    };
    creditNotes: {
        create: (data: Record<string, unknown>) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
    };
    quotes: {
        create: (data: Record<string, unknown>) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
        get: (id: string) => Promise<any>;
        update: (id: string, data: Record<string, unknown>) => Promise<any>;
        send: (id: string) => Promise<any>;
        accept: (id: string) => Promise<any>;
        decline: (id: string) => Promise<any>;
        /** Convert an accepted quote into a subscription. */
        convert: (id: string) => Promise<any>;
        delete: (id: string) => Promise<any>;
    };
    webhooks: {
        /** Register an endpoint to receive event deliveries. */
        create: (data: {
            url: string;
            event_types?: string[];
            [key: string]: unknown;
        }) => Promise<any>;
        list: () => Promise<any>;
        delete: (id: string) => Promise<any>;
        /**
         * Recent delivery attempts to an endpoint, newest first. Filter by
         * derived status (pending | succeeded | failed) and paginate with
         * limit/offset.
         */
        deliveries: (id: string, params?: {
            limit?: number;
            offset?: number;
            status?: "pending" | "succeeded" | "failed";
        }) => Promise<any>;
    };
    events: {
        list: (params?: ListParams) => Promise<any>;
        types: () => Promise<any>;
        /** Delivery attempts of an event across all webhook endpoints. */
        deliveries: (id: string) => Promise<any>;
        /**
         * Re-enqueue delivery of an event to every active subscribed
         * endpoint (202: {event_id, deliveries_queued}). Idempotent.
         */
        redeliver: (id: string) => Promise<any>;
    };
    mandates: {
        create: (data: Record<string, unknown>) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
        get: (id: string) => Promise<any>;
        revoke: (id: string) => Promise<any>;
    };
    gifts: {
        purchase: (data: Record<string, unknown>) => Promise<any>;
        redeem: (data: {
            code: string;
            [key: string]: unknown;
        }) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
    };
    referrals: {
        create: (data: Record<string, unknown>) => Promise<any>;
        list: (params?: ListParams) => Promise<any>;
        generateCode: (data: Record<string, unknown>) => Promise<any>;
        qualify: (id: string) => Promise<any>;
    };
    entitlements: {
        /**
         * Replace a plan's full entitlement set (PUT semantics: feature
         * keys absent from the list are removed).
         */
        setForPlan: (planId: string, list: Array<{
            feature_key: string;
            kind: "boolean" | "limit";
            bool_value?: boolean;
            limit_value?: number;
        }>) => Promise<any>;
        getForPlan: (planId: string) => Promise<any>;
        /**
         * Effective entitlements for a customer: the union over the plans
         * of their active/trialing subscriptions (boolean: any-true wins;
         * limit: max across plans).
         */
        forCustomer: (customerId: string) => Promise<any>;
        /** Fast single-feature check: {feature_key, granted, limit_value}. */
        check: (customerId: string, feature: string) => Promise<any>;
    };
    analytics: {
        /**
         * Monthly recurring revenue, FX-normalized to the tenant's reporting
         * currency: {mrr, normalized_mrr, reporting_currency, breakdown[],
         * fx: {rates, source, as_of}}.
         */
        mrr: () => Promise<any>;
    };
    ledger: {
        accounts: () => Promise<any>;
        entries: (params?: {
            account_id?: string;
            [key: string]: unknown;
        }) => Promise<any>;
    };
}
