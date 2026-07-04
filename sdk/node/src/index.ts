import axios, { AxiosInstance } from 'axios';

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
export class Recurso {
    private client: AxiosInstance;

    constructor(apiKey: string, baseURL: string = 'http://localhost:8080') {
        this.client = axios.create({
            baseURL,
            headers: {
                Authorization: `Bearer ${apiKey}`,
                'Content-Type': 'application/json',
            },
        });
    }

    private get = async (path: string, params?: object) =>
        (await this.client.get(path, { params })).data;
    private post = async (path: string, data?: object) =>
        (await this.client.post(path, data)).data;
    private put = async (path: string, data?: object) =>
        (await this.client.put(path, data)).data;
    private del = async (path: string) =>
        (await this.client.delete(path)).data;

    public account = {
        get: () => this.get('/v1/account'),
        update: (data: Record<string, unknown>) => this.put('/v1/account', data),
    };

    public customers = {
        create: (data: {
            email: string;
            name: string;
            line1?: string;
            city?: string;
            state?: string;
            zip?: string;
            country?: string;
            [key: string]: unknown;
        }) => this.post('/v1/customers', { country: 'US', ...data }),
        list: (params?: ListParams) => this.get('/v1/customers', params),
        updatePaymentMethod: (id: string, data: Record<string, unknown>) =>
            this.put(`/v1/customers/${id}/payment-method`, data),
        churn: (id: string) => this.get(`/v1/customers/${id}/churn`),
        consents: (id: string) => this.get(`/v1/customers/${id}/consents`),
    };

    public plans = {
        create: (data: {
            name: string;
            code: string;
            amount: number;
            currency: string;
            interval_unit: 'day' | 'week' | 'month' | 'year';
            interval_count?: number;
            [key: string]: unknown;
        }) => this.post('/v1/plans', { interval_count: 1, ...data }),
        list: (params?: ListParams) => this.get('/v1/plans', params),
    };

    public subscriptions = {
        create: (data: {
            customer_id: string;
            plan_id: string;
            coupon_code?: string;
            start_date?: string;
            payment_terms?: string;
            [key: string]: unknown;
        }) => this.post('/v1/subscriptions', data),
        list: (params?: ListParams) => this.get('/v1/subscriptions', params),
        update: (id: string, data: Record<string, unknown>) =>
            this.put(`/v1/subscriptions/${id}`, data),
        cancel: (id: string, data?: Record<string, unknown>) =>
            this.post(`/v1/subscriptions/${id}/cancel`, data),
        pause: (id: string, data?: Record<string, unknown>) =>
            this.post(`/v1/subscriptions/${id}/pause`, data),
        resume: (id: string) => this.post(`/v1/subscriptions/${id}/resume`),
        reactivate: (id: string) => this.post(`/v1/subscriptions/${id}/reactivate`),
        /** Bill N future periods immediately (advance invoicing). */
        advance: (id: string, data: { periods: number }) =>
            this.post(`/v1/subscriptions/${id}/advance`, data),
        charges: (id: string) => this.get(`/v1/subscriptions/${id}/charges`),
        addCharge: (id: string, data: Record<string, unknown>) =>
            this.post(`/v1/subscriptions/${id}/charges`, data),
    };

    public invoices = {
        list: (params?: ListParams) => this.get('/v1/invoices', params),
        /** Public PDF download URL for an invoice. */
        pdfUrl: (id: string) =>
            `${this.client.defaults.baseURL}/v1/invoices/${id}/pdf`,
        eInvoiceStatus: (id: string) => this.get(`/v1/invoices/${id}/einvoice`),
        retryEInvoice: (id: string) => this.post(`/v1/invoices/${id}/einvoice/retry`),
        cancelEInvoice: (id: string, data?: Record<string, unknown>) =>
            this.post(`/v1/invoices/${id}/einvoice/cancel`, data),
    };

    public coupons = {
        create: (data: {
            code: string;
            discount_type: 'percent' | 'amount';
            discount_value: number;
            duration: 'forever' | 'once';
            [key: string]: unknown;
        }) => this.post('/v1/coupons', data),
        list: (params?: ListParams) => this.get('/v1/coupons', params),
    };

    public usage = {
        /** Record a metered usage event against a subscription. */
        record: (data: {
            subscription_id: string;
            customer_id: string;
            dimension: string;
            quantity: number;
        }) => this.post('/v1/usage/events', data),
    };

    public creditNotes = {
        create: (data: Record<string, unknown>) => this.post('/v1/credit-notes', data),
        list: (params?: ListParams) => this.get('/v1/credit-notes', params),
    };

    public quotes = {
        create: (data: Record<string, unknown>) => this.post('/v1/quotes', data),
        list: (params?: ListParams) => this.get('/v1/quotes', params),
        get: (id: string) => this.get(`/v1/quotes/${id}`),
        update: (id: string, data: Record<string, unknown>) =>
            this.put(`/v1/quotes/${id}`, data),
        send: (id: string) => this.post(`/v1/quotes/${id}/send`),
        accept: (id: string) => this.post(`/v1/quotes/${id}/accept`),
        decline: (id: string) => this.post(`/v1/quotes/${id}/decline`),
        /** Convert an accepted quote into a subscription. */
        convert: (id: string) => this.post(`/v1/quotes/${id}/convert`),
        delete: (id: string) => this.del(`/v1/quotes/${id}`),
    };

    public webhooks = {
        /** Register an endpoint to receive event deliveries. */
        create: (data: { url: string; event_types?: string[]; [key: string]: unknown }) =>
            this.post('/v1/webhooks', data),
        list: () => this.get('/v1/webhooks'),
        delete: (id: string) => this.del(`/v1/webhooks/${id}`),
    };

    public events = {
        list: (params?: ListParams) => this.get('/v1/events', params),
        types: () => this.get('/v1/events/types'),
    };

    public mandates = {
        create: (data: Record<string, unknown>) => this.post('/v1/mandates', data),
        list: (params?: ListParams) => this.get('/v1/mandates', params),
        get: (id: string) => this.get(`/v1/mandates/${id}`),
        revoke: (id: string) => this.post(`/v1/mandates/${id}/revoke`),
    };

    public gifts = {
        purchase: (data: Record<string, unknown>) => this.post('/v1/gifts/purchase', data),
        redeem: (data: { code: string; [key: string]: unknown }) =>
            this.post('/v1/gifts/redeem', data),
        list: (params?: ListParams) => this.get('/v1/gifts', params),
    };

    public referrals = {
        create: (data: Record<string, unknown>) => this.post('/v1/referrals', data),
        list: (params?: ListParams) => this.get('/v1/referrals', params),
        generateCode: (data: Record<string, unknown>) =>
            this.post('/v1/referrals/generate-code', data),
        qualify: (id: string) => this.post(`/v1/referrals/${id}/qualify`),
    };

    public ledger = {
        accounts: () => this.get('/v1/ledger/accounts'),
        entries: (params?: { account_id?: string; [key: string]: unknown }) =>
            this.get('/v1/ledger/entries', params),
    };
}
