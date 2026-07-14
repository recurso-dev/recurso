/** Default API base URL, used when none is supplied at construction. */
export declare const DEFAULT_BASE_URL = "http://localhost:8080";
/** Options accepted by the {@link Recurso} constructor. */
export interface RecursoOptions {
    /** API base URL. Defaults to {@link DEFAULT_BASE_URL} when omitted. */
    baseUrl?: string;
}
/** Common list-endpoint query parameters (all optional, server-side). */
export interface ListParams {
    page?: number;
    limit?: number;
    q?: string;
    status?: string;
    [key: string]: unknown;
}
/**
 * A JSON value returned by (or sent to) the API. The API speaks JSON, so any
 * payload or response is one of these shapes.
 */
export type JsonValue = string | number | boolean | null | JsonValue[] | {
    [key: string]: JsonValue;
};
/** A JSON object body — the shape of every request payload and object response. */
export type JsonObject = {
    [key: string]: JsonValue;
};
/**
 * Generic API response. Endpoints return either a single resource object or a
 * paginated envelope; both are JSON objects, so callers can narrow as needed.
 */
export type ApiResponse = JsonObject;
/** Paginated list envelope returned by `list` endpoints. */
export interface ListResponse<T = JsonObject> {
    data: T[];
    page?: number;
    limit?: number;
    total?: number;
    [key: string]: JsonValue | T[] | undefined;
}
/** Payload for creating or updating a customer. */
export interface CustomerInput {
    email: string;
    name: string;
    line1?: string;
    city?: string;
    state?: string;
    zip?: string;
    country?: string;
    [key: string]: JsonValue | undefined;
}
/** Payload for creating a plan. */
export interface PlanInput {
    name: string;
    code: string;
    amount: number;
    currency: string;
    interval_unit: 'day' | 'week' | 'month' | 'year';
    interval_count?: number;
    [key: string]: JsonValue | undefined;
}
/** Payload for creating a subscription. */
export interface SubscriptionInput {
    customer_id: string;
    plan_id: string;
    coupon_code?: string;
    start_date?: string;
    payment_terms?: string;
    [key: string]: JsonValue | undefined;
}
/** Payload for creating a coupon. */
export interface CouponInput {
    code: string;
    discount_type: 'percent' | 'amount';
    discount_value: number;
    duration: 'forever' | 'once';
    [key: string]: JsonValue | undefined;
}
/** Payload for recording a metered usage event. */
export interface UsageEventInput {
    subscription_id: string;
    customer_id: string;
    dimension: string;
    quantity: number;
}
/** Query parameters for the time-windowed usage endpoint. */
export interface UsageQueryParams {
    subscription_id?: string;
    customer_id?: string;
    dimension?: string;
    from?: string;
    to?: string;
    granularity?: 'day' | 'month';
}
/** Payload for registering a webhook endpoint. */
export interface WebhookInput {
    url: string;
    event_types?: string[];
    [key: string]: JsonValue | undefined;
}
/** Query parameters for listing webhook deliveries. */
export interface WebhookDeliveriesParams {
    limit?: number;
    offset?: number;
    status?: 'pending' | 'succeeded' | 'failed';
}
/** Payload for redeeming a gift. */
export interface GiftRedeemInput {
    code: string;
    [key: string]: JsonValue | undefined;
}
/** A single entitlement in a plan's entitlement set. */
export interface Entitlement {
    feature_key: string;
    kind: 'boolean' | 'limit';
    bool_value?: boolean;
    limit_value?: number;
}
/** Query parameters for listing ledger entries. */
export interface LedgerEntriesParams {
    account_id?: string;
    [key: string]: JsonValue | undefined;
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
    /**
     * @param apiKey  API key used for Bearer authentication.
     * @param options Client options. Pass `{ baseUrl }` to target a specific
     *                environment; a bare string is also accepted for backward
     *                compatibility. Defaults to {@link DEFAULT_BASE_URL}.
     */
    constructor(apiKey: string, options?: string | RecursoOptions);
    private get;
    private post;
    private put;
    private del;
    account: {
        get: () => Promise<JsonObject>;
        update: (data: JsonObject) => Promise<JsonObject>;
    };
    customers: {
        create: (data: CustomerInput) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
        updatePaymentMethod: (id: string, data: JsonObject) => Promise<JsonObject>;
        churn: (id: string) => Promise<JsonObject>;
        consents: (id: string) => Promise<JsonObject>;
    };
    plans: {
        create: (data: PlanInput) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
    };
    subscriptions: {
        create: (data: SubscriptionInput) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
        update: (id: string, data: JsonObject) => Promise<JsonObject>;
        cancel: (id: string, data?: JsonObject) => Promise<JsonObject>;
        pause: (id: string, data?: JsonObject) => Promise<JsonObject>;
        resume: (id: string) => Promise<JsonObject>;
        reactivate: (id: string) => Promise<JsonObject>;
        /** Bill N future periods immediately (advance invoicing). */
        advance: (id: string, data: {
            periods: number;
        }) => Promise<JsonObject>;
        charges: (id: string) => Promise<JsonObject>;
        addCharge: (id: string, data: JsonObject) => Promise<JsonObject>;
        /**
         * Current billing period's usage per dimension plus lifetime
         * totals, with the customer's entitlement limit/remaining joined
         * in where a feature_key matches the dimension name.
         */
        usage: (id: string) => Promise<JsonObject>;
    };
    invoices: {
        list: (params?: ListParams) => Promise<JsonObject>;
        /** Public PDF download URL for an invoice. */
        pdfUrl: (id: string) => string;
        eInvoiceStatus: (id: string) => Promise<JsonObject>;
        retryEInvoice: (id: string) => Promise<JsonObject>;
        cancelEInvoice: (id: string, data?: JsonObject) => Promise<JsonObject>;
    };
    coupons: {
        create: (data: CouponInput) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
    };
    usage: {
        /** Record a metered usage event against a subscription. */
        record: (data: UsageEventInput) => Promise<JsonObject>;
        /**
         * Time-windowed usage buckets: {data: [{period, dimension,
         * quantity}], from, to, granularity}. At least one of
         * subscription_id or customer_id is required; the window defaults
         * to the last 30 days at day granularity.
         */
        query: (params: UsageQueryParams) => Promise<JsonObject>;
        /** The tenant's dimension catalog with first/last seen and event counts. */
        dimensions: () => Promise<JsonObject>;
    };
    creditNotes: {
        create: (data: JsonObject) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
    };
    quotes: {
        create: (data: JsonObject) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
        get: (id: string) => Promise<JsonObject>;
        update: (id: string, data: JsonObject) => Promise<JsonObject>;
        send: (id: string) => Promise<JsonObject>;
        accept: (id: string) => Promise<JsonObject>;
        decline: (id: string) => Promise<JsonObject>;
        /** Convert an accepted quote into a subscription. */
        convert: (id: string) => Promise<JsonObject>;
        delete: (id: string) => Promise<JsonObject>;
    };
    webhooks: {
        /** Register an endpoint to receive event deliveries. */
        create: (data: WebhookInput) => Promise<JsonObject>;
        list: () => Promise<JsonObject>;
        delete: (id: string) => Promise<JsonObject>;
        /**
         * Recent delivery attempts to an endpoint, newest first. Filter by
         * derived status (pending | succeeded | failed) and paginate with
         * limit/offset.
         */
        deliveries: (id: string, params?: WebhookDeliveriesParams) => Promise<JsonObject>;
    };
    events: {
        list: (params?: ListParams) => Promise<JsonObject>;
        types: () => Promise<JsonObject>;
        /** Delivery attempts of an event across all webhook endpoints. */
        deliveries: (id: string) => Promise<JsonObject>;
        /**
         * Re-enqueue delivery of an event to every active subscribed
         * endpoint (202: {event_id, deliveries_queued}). Idempotent.
         */
        redeliver: (id: string) => Promise<JsonObject>;
    };
    mandates: {
        create: (data: JsonObject) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
        get: (id: string) => Promise<JsonObject>;
        revoke: (id: string) => Promise<JsonObject>;
    };
    gifts: {
        purchase: (data: JsonObject) => Promise<JsonObject>;
        redeem: (data: GiftRedeemInput) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
    };
    referrals: {
        create: (data: JsonObject) => Promise<JsonObject>;
        list: (params?: ListParams) => Promise<JsonObject>;
        generateCode: (data: JsonObject) => Promise<JsonObject>;
        qualify: (id: string) => Promise<JsonObject>;
    };
    entitlements: {
        /**
         * Replace a plan's full entitlement set (PUT semantics: feature
         * keys absent from the list are removed).
         */
        setForPlan: (planId: string, list: Entitlement[]) => Promise<JsonObject>;
        getForPlan: (planId: string) => Promise<JsonObject>;
        /**
         * Effective entitlements for a customer: the union over the plans
         * of their active/trialing subscriptions (boolean: any-true wins;
         * limit: max across plans).
         */
        forCustomer: (customerId: string) => Promise<JsonObject>;
        /** Fast single-feature check: {feature_key, granted, limit_value}. */
        check: (customerId: string, feature: string) => Promise<JsonObject>;
    };
    analytics: {
        /**
         * Monthly recurring revenue, FX-normalized to the tenant's reporting
         * currency: {mrr, normalized_mrr, reporting_currency, breakdown[],
         * fx: {rates, source, as_of}}.
         */
        mrr: () => Promise<JsonObject>;
    };
    ledger: {
        accounts: () => Promise<JsonObject>;
        entries: (params?: LedgerEntriesParams) => Promise<JsonObject>;
    };
}
