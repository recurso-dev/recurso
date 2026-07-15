import type { components, operations } from './schema';
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
 * Generic API response. Used only as the fallback for endpoints the spec does
 * not model; typed endpoints resolve to their concrete resource shape instead.
 */
export type ApiResponse = JsonObject;
/** Every resource/request model defined by the API, keyed by schema name. */
export type Schemas = components['schemas'];
export type Customer = Schemas['Customer'];
export type Plan = Schemas['Plan'];
export type Price = Schemas['Price'];
export type Subscription = Schemas['Subscription'];
export type UnbilledCharge = Schemas['UnbilledCharge'];
export type SubscriptionUsage = Schemas['SubscriptionUsage'];
export type Invoice = Schemas['Invoice'];
export type Coupon = Schemas['Coupon'];
export type CreditNote = Schemas['CreditNote'];
export type Quote = Schemas['Quote'];
export type QuoteActionResponse = Schemas['QuoteActionResponse'];
export type WebhookEndpoint = Schemas['WebhookEndpoint'];
export type Event = Schemas['Event'];
export type EventDelivery = Schemas['EventDelivery'];
export type Mandate = Schemas['Mandate'];
export type Gift = Schemas['Gift'];
export type Referral = Schemas['Referral'];
export type MRRMetrics = Schemas['MRRMetrics'];
export type LedgerAccount = Schemas['LedgerAccount'];
export type LedgerTransaction = Schemas['LedgerTransaction'];
export type ChurnScoreResult = Schemas['ChurnScoreResult'];
export type Consent = Schemas['Consent'];
export type Tenant = Schemas['Tenant'];
/** The JSON body of an operation's success (2xx) response, per the spec. */
type SuccessJson<O> = O extends {
    responses: infer R;
} ? R extends {
    200: {
        content: {
            'application/json': infer B;
        };
    };
} ? B : R extends {
    201: {
        content: {
            'application/json': infer B;
        };
    };
} ? B : R extends {
    202: {
        content: {
            'application/json': infer B;
        };
    };
} ? B : ApiResponse : ApiResponse;
/** Response body type for a named operation. */
export type Res<K extends keyof operations> = SuccessJson<operations[K]>;
/** The JSON request body of an operation, if it defines one. */
type RequestJson<O> = O extends {
    requestBody?: {
        content: {
            'application/json': infer B;
        };
    };
} ? B : never;
/** Request body type for a named operation. */
export type Body<K extends keyof operations> = RequestJson<operations[K]>;
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
 * (cancel, pause, resume, ...) live on their resource. Return types are
 * derived from the OpenAPI spec, so responses carry full field-level types.
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
        get: () => Promise<{
            data?: components["schemas"]["Tenant"];
        }>;
        update: (data: Body<"updateAccount">) => Promise<{
            data?: components["schemas"]["Tenant"];
        }>;
    };
    customers: {
        create: (data: CustomerInput) => Promise<{
            id?: string;
            tenant_id?: string;
            email?: string;
            name?: string | null;
            phone?: string;
            tax_id?: string | null;
            billing_address?: components["schemas"]["BillingAddress"];
            ledger_account_id?: string;
            gstin?: string | null;
            tax_type?: string;
            place_of_supply?: string | null;
            referral_code?: string | null;
            risk_score?: number;
            risk_factors?: Record<string, never> | null;
            card_brand?: string;
            card_last4?: string;
            card_exp_month?: number;
            card_exp_year?: number;
            card_token_id?: string;
            card_fingerprint?: string;
            created_at?: string;
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Customer"][];
        }>;
        updatePaymentMethod: (id: string, data: Body<"updateCustomerPaymentMethod">) => Promise<{
            status?: "ok";
        }>;
        churn: (id: string) => Promise<{
            data?: components["schemas"]["ChurnScoreResult"];
        }>;
        consents: (id: string) => Promise<{
            object?: "list";
            data?: components["schemas"]["Consent"][];
        }>;
    };
    plans: {
        create: (data: PlanInput) => Promise<{
            id?: string;
            tenant_id?: string;
            name?: string;
            code?: string;
            interval_unit?: "day" | "week" | "month" | "year";
            interval_count?: number;
            active?: boolean;
            hsn_code?: string;
            created_at?: string;
            prices?: components["schemas"]["Price"][];
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Plan"][];
        }>;
    };
    subscriptions: {
        create: (data: SubscriptionInput) => Promise<{
            id?: string;
            tenant_id?: string;
            customer_id?: string;
            plan_id?: string;
            status?: components["schemas"]["SubscriptionStatus"];
            current_period_start?: string;
            current_period_end?: string;
            cancel_at_period_end?: boolean;
            canceled_at?: string;
            cancellation_reason?: string;
            cancellation_feedback?: string;
            billing_anchor?: string;
            billing_anchor_type?: string;
            billing_anchor_day?: number;
            payment_terms?: string;
            coupon_id?: string;
            reference_id?: string;
            mandate_id?: string;
            razorpay_subscription_id?: string;
            stripe_subscription_id?: string;
            created_at?: string;
            updated_at?: string;
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Subscription"][];
        }>;
        update: (id: string, data: Body<"updateSubscription">) => Promise<{
            id?: string;
            tenant_id?: string;
            customer_id?: string;
            plan_id?: string;
            status?: components["schemas"]["SubscriptionStatus"];
            current_period_start?: string;
            current_period_end?: string;
            cancel_at_period_end?: boolean;
            canceled_at?: string;
            cancellation_reason?: string;
            cancellation_feedback?: string;
            billing_anchor?: string;
            billing_anchor_type?: string;
            billing_anchor_day?: number;
            payment_terms?: string;
            coupon_id?: string;
            reference_id?: string;
            mandate_id?: string;
            razorpay_subscription_id?: string;
            stripe_subscription_id?: string;
            created_at?: string;
            updated_at?: string;
        }>;
        cancel: (id: string, data?: Body<"cancelSubscription">) => Promise<{
            id?: string;
            status?: string;
            cancel_at_period_end?: boolean;
            cancelled_at?: string;
            current_period_end?: string;
            cancellation_reason?: string;
            message?: string;
        }>;
        pause: (id: string, data?: Body<"pauseSubscription">) => Promise<{
            data?: components["schemas"]["Subscription"];
        }>;
        resume: (id: string) => Promise<{
            data?: components["schemas"]["Subscription"];
        }>;
        reactivate: (id: string) => Promise<{
            id?: string;
            status?: string;
            message?: string;
        }>;
        /** Bill N future periods immediately (advance invoicing). */
        advance: (id: string, data: Body<"generateAdvanceInvoice">) => Promise<{
            id?: string;
            tenant_id?: string;
            subscription_id?: string | null;
            customer_id?: string;
            invoice_number?: string;
            billing_reason?: string;
            amount_due?: number;
            amount_paid?: number;
            currency?: string;
            subtotal?: number;
            tax_amount?: number;
            total?: number;
            igst_amount?: number;
            cgst_amount?: number;
            sgst_amount?: number;
            hsn_code?: string;
            irn?: string;
            ack_no?: string;
            signed_qr_code?: string;
            e_invoice_status?: string;
            ack_date?: string;
            e_invoice_retry_count?: number;
            e_invoice_next_retry_at?: string | null;
            e_invoice_error_message?: string;
            tds_amount?: number;
            status?: "draft" | "open" | "paid" | "void" | "uncollectible" | "past_due";
            created_at?: string;
            due_date?: string;
            paid_at?: string;
            payment_terms?: string;
            exchange_rate?: number;
            base_currency_total?: number;
            base_currency?: string;
            next_retry_at?: string;
            retry_count?: number;
            payment_wall_active?: boolean;
            line_items?: components["schemas"]["InvoiceItem"][];
        }>;
        charges: (id: string) => Promise<{
            data?: components["schemas"]["UnbilledCharge"][];
        }>;
        addCharge: (id: string, data: Body<"addUnbilledCharge">) => Promise<{
            id?: string;
            subscription_id?: string;
            amount?: number;
            currency?: string;
            description?: string;
            hsn_code?: string;
            status?: "pending" | "invoiced" | "canceled";
            period_start?: string;
            period_end?: string;
            created_at?: string;
        }>;
        /**
         * Current billing period's usage per dimension plus lifetime
         * totals, with the customer's entitlement limit/remaining joined
         * in where a feature_key matches the dimension name.
         */
        usage: (id: string) => Promise<{
            subscription_id?: string;
            customer_id?: string;
            current_period_start?: string;
            current_period_end?: string;
            dimensions?: components["schemas"]["SubscriptionDimensionUsage"][];
        }>;
    };
    invoices: {
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Invoice"][];
        }>;
        /** Public PDF download URL for an invoice. */
        pdfUrl: (id: string) => string;
        eInvoiceStatus: (id: string) => Promise<{
            data?: components["schemas"]["EInvoiceStatus"];
        }>;
        retryEInvoice: (id: string) => Promise<{
            data?: Record<string, never>;
            message?: string;
        }>;
        cancelEInvoice: (id: string, data?: Body<"cancelEInvoice">) => Promise<{
            message?: string;
        }>;
    };
    coupons: {
        create: (data: CouponInput) => Promise<{
            id?: string;
            tenant_id?: string;
            code?: string;
            discount_type?: "percent" | "amount";
            discount_value?: number;
            duration?: "forever" | "once" | "repeating";
            duration_months?: number | null;
            created_at?: string;
            updated_at?: string;
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Coupon"][];
        }>;
    };
    usage: {
        /** Record a metered usage event against a subscription. */
        record: (data: UsageEventInput) => Promise<{
            status?: "recorded";
            event_id?: string;
        }>;
        /**
         * Time-windowed usage buckets: {data: [{period, dimension,
         * quantity}], from, to, granularity}. At least one of
         * subscription_id or customer_id is required; the window defaults
         * to the last 30 days at day granularity.
         */
        query: (params: UsageQueryParams) => Promise<{
            data?: components["schemas"]["UsageBucket"][];
            from?: string;
            to?: string;
            granularity?: "day" | "month";
        }>;
        /** The tenant's dimension catalog with first/last seen and event counts. */
        dimensions: () => Promise<{
            data?: components["schemas"]["UsageDimension"][];
        }>;
    };
    creditNotes: {
        create: (data: Body<"createCreditNote">) => Promise<{
            data?: components["schemas"]["CreditNote"];
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["CreditNote"][];
        }>;
    };
    quotes: {
        create: (data: Body<"createQuote">) => Promise<{
            data?: components["schemas"]["Quote"];
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Quote"][];
        }>;
        get: (id: string) => Promise<{
            data?: components["schemas"]["Quote"];
        }>;
        update: (id: string, data: Body<"updateQuote">) => Promise<{
            data?: components["schemas"]["Quote"];
        }>;
        send: (id: string) => Promise<{
            data?: components["schemas"]["Quote"];
            message?: string;
        }>;
        accept: (id: string) => Promise<{
            data?: components["schemas"]["Quote"];
            message?: string;
        }>;
        decline: (id: string) => Promise<{
            data?: components["schemas"]["Quote"];
            message?: string;
        }>;
        /** Convert an accepted quote into a subscription. */
        convert: (id: string) => Promise<{
            data?: components["schemas"]["Invoice"];
            message?: string;
        }>;
        delete: (id: string) => Promise<{
            message?: string;
        }>;
    };
    webhooks: {
        /** Register an endpoint to receive event deliveries. */
        create: (data: WebhookInput) => Promise<{
            data?: components["schemas"]["WebhookEndpoint"];
        }>;
        list: () => Promise<{
            data?: components["schemas"]["WebhookEndpoint"][];
        }>;
        delete: (id: string) => Promise<{
            message?: string;
        }>;
        /**
         * Recent delivery attempts to an endpoint, newest first. Filter by
         * derived status (pending | succeeded | failed) and paginate with
         * limit/offset.
         */
        deliveries: (id: string, params?: WebhookDeliveriesParams) => Promise<{
            data?: components["schemas"]["EventDelivery"][];
        }>;
    };
    events: {
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Event"][];
        }>;
        types: () => Promise<{
            data?: string[];
        }>;
        /** Delivery attempts of an event across all webhook endpoints. */
        deliveries: (id: string) => Promise<{
            data?: components["schemas"]["EventDelivery"][];
        }>;
        /**
         * Re-enqueue delivery of an event to every active subscribed
         * endpoint (202: {event_id, deliveries_queued}). Idempotent.
         */
        redeliver: (id: string) => Promise<{
            data?: {
                event_id?: string;
                deliveries_queued?: number;
            };
        }>;
    };
    mandates: {
        create: (data: Body<"createMandate">) => Promise<{
            mandate?: components["schemas"]["Mandate"];
            auth_url?: string;
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Mandate"][];
        }>;
        get: (id: string) => Promise<{
            data?: components["schemas"]["Mandate"];
        }>;
        revoke: (id: string) => Promise<{
            status?: "revoked";
        }>;
    };
    gifts: {
        purchase: (data: Body<"purchaseGift">) => Promise<{
            id?: string;
            tenant_id?: string;
            code?: string;
            plan_id?: string;
            buyer_customer_id?: string;
            recipient_email?: string;
            status?: "purchased" | "redeemed";
            redeemed_by_customer_id?: string | null;
            redeemed_at?: string | null;
            duration_months?: number;
            created_at?: string;
            updated_at?: string;
        }>;
        redeem: (data: GiftRedeemInput) => Promise<{
            id?: string;
            tenant_id?: string;
            customer_id?: string;
            plan_id?: string;
            status?: components["schemas"]["SubscriptionStatus"];
            current_period_start?: string;
            current_period_end?: string;
            cancel_at_period_end?: boolean;
            canceled_at?: string;
            cancellation_reason?: string;
            cancellation_feedback?: string;
            billing_anchor?: string;
            billing_anchor_type?: string;
            billing_anchor_day?: number;
            payment_terms?: string;
            coupon_id?: string;
            reference_id?: string;
            mandate_id?: string;
            razorpay_subscription_id?: string;
            stripe_subscription_id?: string;
            created_at?: string;
            updated_at?: string;
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Gift"][];
            meta?: components["schemas"]["PageMeta"];
        }>;
    };
    referrals: {
        create: (data: Body<"createReferral">) => Promise<{
            data?: components["schemas"]["Referral"];
        }>;
        list: (params?: ListParams) => Promise<{
            data?: components["schemas"]["Referral"][];
            meta?: components["schemas"]["PageMeta"];
        }>;
        generateCode: (data: Body<"generateReferralCode">) => Promise<{
            data?: {
                code?: string;
            };
        }>;
        qualify: (id: string) => Promise<{
            data?: components["schemas"]["Referral"];
        }>;
    };
    entitlements: {
        /**
         * Replace a plan's full entitlement set (PUT semantics: feature
         * keys absent from the list are removed).
         */
        setForPlan: (planId: string, list: Entitlement[]) => Promise<{
            data?: components["schemas"]["Entitlement"][];
        }>;
        getForPlan: (planId: string) => Promise<{
            data?: components["schemas"]["Entitlement"][];
        }>;
        /**
         * Effective entitlements for a customer: the union over the plans
         * of their active/trialing subscriptions (boolean: any-true wins;
         * limit: max across plans).
         */
        forCustomer: (customerId: string) => Promise<{
            data?: {
                feature_key?: string;
                kind?: "boolean" | "limit";
                bool_value?: boolean | null;
                limit_value?: number | null;
                plan_ids?: string[];
            }[];
        }>;
        /** Fast single-feature check: {feature_key, granted, limit_value}. */
        check: (customerId: string, feature: string) => Promise<{
            feature_key?: string;
            granted?: boolean;
            limit_value?: number | null;
        }>;
    };
    analytics: {
        /**
         * Monthly recurring revenue, FX-normalized to the tenant's reporting
         * currency: {mrr, normalized_mrr, reporting_currency, breakdown[],
         * fx: {rates, source, as_of}}.
         */
        mrr: () => Promise<{
            currency?: string;
            amount?: number;
            mrr?: number;
            normalized_mrr?: number;
            reporting_currency?: string;
            breakdown?: components["schemas"]["MRRCurrencyBreakdown"][];
            fx?: components["schemas"]["FXSnapshot"];
        }>;
    };
    ledger: {
        accounts: () => Promise<{
            data?: components["schemas"]["LedgerAccount"][];
        }>;
        entries: (params?: LedgerEntriesParams) => Promise<{
            data?: components["schemas"]["LedgerTransaction"][];
        }>;
    };
}
export {};
