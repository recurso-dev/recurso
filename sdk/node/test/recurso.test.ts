import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Recurso } from '../src/index';
import axios from 'axios';

/**
 * The SDK is a thin typed wrapper over one axios instance, so the whole
 * suite runs against a mocked `axios.create()` that returns a capture
 * client. Every resource method is exercised from a table asserting:
 * verb, path (incl. path params), query/body passthrough, and that the
 * axios response envelope is unwrapped to `.data`.
 */

const mocks = vi.hoisted(() => {
    const client = {
        get: vi.fn(),
        post: vi.fn(),
        put: vi.fn(),
        delete: vi.fn(),
        defaults: { baseURL: undefined as string | undefined },
    };
    const create = vi.fn((config?: { baseURL?: string }) => {
        client.defaults.baseURL = config?.baseURL;
        return client;
    });
    return { client, create };
});

vi.mock('axios', () => ({ default: { create: mocks.create } }));

/** Sentinel payload: proves `.data` unwrapping, not just call success. */
const PAYLOAD = { id: 'obj_123', unwrapped: true };
const RESPONSE = { status: 200, data: PAYLOAD };

const API_KEY = 'rsk_test_abc123';
const BASE_URL = 'https://billing.example.com';

let recurso: Recurso;

beforeEach(() => {
    mocks.create.mockClear();
    for (const verb of ['get', 'post', 'put', 'delete'] as const) {
        mocks.client[verb].mockReset().mockResolvedValue(RESPONSE);
    }
    recurso = new Recurso(API_KEY, BASE_URL);
});

type Verb = 'get' | 'post' | 'put' | 'delete';

interface MethodCase {
    /** Method name, for the subtest title. */
    method: string;
    /** Invokes the SDK method under test. */
    call: (r: Recurso) => Promise<unknown>;
    verb: Verb;
    path: string;
    /** Expected query params object (GET) — omit for "no params passed". */
    params?: object;
    /** Expected request body (POST/PUT) — omit for "no body passed". */
    body?: unknown;
}

const listParams = { page: 2, limit: 50, q: 'search', status: 'active' };
const body = { any: 'payload', nested: { n: 1 } };

const cases: Record<string, MethodCase[]> = {
    account: [
        { method: 'get', call: (r) => r.account.get(), verb: 'get', path: '/v1/account' },
        { method: 'update', call: (r) => r.account.update(body), verb: 'put', path: '/v1/account', body },
    ],

    customers: [
        {
            method: 'create',
            call: (r) => r.customers.create({ email: 'jane@example.com', name: 'Jane', country: 'IN' }),
            verb: 'post',
            path: '/v1/customers',
            body: { email: 'jane@example.com', name: 'Jane', country: 'IN' },
        },
        { method: 'list', call: (r) => r.customers.list(listParams), verb: 'get', path: '/v1/customers', params: listParams },
        {
            method: 'updatePaymentMethod',
            call: (r) => r.customers.updatePaymentMethod('cus_1', body),
            verb: 'put',
            path: '/v1/customers/cus_1/payment-method',
            body,
        },
        { method: 'churn', call: (r) => r.customers.churn('cus_1'), verb: 'get', path: '/v1/customers/cus_1/churn' },
        { method: 'consents', call: (r) => r.customers.consents('cus_1'), verb: 'get', path: '/v1/customers/cus_1/consents' },
    ],

    plans: [
        {
            method: 'create',
            call: (r) =>
                r.plans.create({ name: 'Pro', code: 'PRO', amount: 2900, currency: 'USD', interval_unit: 'month', interval_count: 3 }),
            verb: 'post',
            path: '/v1/plans',
            body: { name: 'Pro', code: 'PRO', amount: 2900, currency: 'USD', interval_unit: 'month', interval_count: 3 },
        },
        { method: 'list', call: (r) => r.plans.list(listParams), verb: 'get', path: '/v1/plans', params: listParams },
    ],

    subscriptions: [
        {
            method: 'create',
            call: (r) => r.subscriptions.create({ customer_id: 'cus_1', plan_id: 'plan_1', coupon_code: 'SAVE10' }),
            verb: 'post',
            path: '/v1/subscriptions',
            body: { customer_id: 'cus_1', plan_id: 'plan_1', coupon_code: 'SAVE10' },
        },
        { method: 'list', call: (r) => r.subscriptions.list(listParams), verb: 'get', path: '/v1/subscriptions', params: listParams },
        { method: 'update', call: (r) => r.subscriptions.update('sub_1', body), verb: 'put', path: '/v1/subscriptions/sub_1', body },
        {
            method: 'cancel',
            call: (r) => r.subscriptions.cancel('sub_1', { at_period_end: true }),
            verb: 'post',
            path: '/v1/subscriptions/sub_1/cancel',
            body: { at_period_end: true },
        },
        {
            method: 'pause',
            call: (r) => r.subscriptions.pause('sub_1', { resume_at: '2026-08-01' }),
            verb: 'post',
            path: '/v1/subscriptions/sub_1/pause',
            body: { resume_at: '2026-08-01' },
        },
        { method: 'resume', call: (r) => r.subscriptions.resume('sub_1'), verb: 'post', path: '/v1/subscriptions/sub_1/resume' },
        {
            method: 'reactivate',
            call: (r) => r.subscriptions.reactivate('sub_1'),
            verb: 'post',
            path: '/v1/subscriptions/sub_1/reactivate',
        },
        {
            method: 'advance',
            call: (r) => r.subscriptions.advance('sub_1', { periods: 3 }),
            verb: 'post',
            path: '/v1/subscriptions/sub_1/advance',
            body: { periods: 3 },
        },
        { method: 'charges', call: (r) => r.subscriptions.charges('sub_1'), verb: 'get', path: '/v1/subscriptions/sub_1/charges' },
        {
            method: 'addCharge',
            call: (r) => r.subscriptions.addCharge('sub_1', body),
            verb: 'post',
            path: '/v1/subscriptions/sub_1/charges',
            body,
        },
    ],

    invoices: [
        { method: 'list', call: (r) => r.invoices.list(listParams), verb: 'get', path: '/v1/invoices', params: listParams },
        { method: 'eInvoiceStatus', call: (r) => r.invoices.eInvoiceStatus('inv_1'), verb: 'get', path: '/v1/invoices/inv_1/einvoice' },
        {
            method: 'retryEInvoice',
            call: (r) => r.invoices.retryEInvoice('inv_1'),
            verb: 'post',
            path: '/v1/invoices/inv_1/einvoice/retry',
        },
        {
            method: 'cancelEInvoice',
            call: (r) => r.invoices.cancelEInvoice('inv_1', { reason: 'data entry error' }),
            verb: 'post',
            path: '/v1/invoices/inv_1/einvoice/cancel',
            body: { reason: 'data entry error' },
        },
    ],

    coupons: [
        {
            method: 'create',
            call: (r) => r.coupons.create({ code: 'SAVE10', discount_type: 'percent', discount_value: 10, duration: 'forever' }),
            verb: 'post',
            path: '/v1/coupons',
            body: { code: 'SAVE10', discount_type: 'percent', discount_value: 10, duration: 'forever' },
        },
        { method: 'list', call: (r) => r.coupons.list(listParams), verb: 'get', path: '/v1/coupons', params: listParams },
    ],

    usage: [
        {
            method: 'record',
            call: (r) => r.usage.record({ subscription_id: 'sub_1', customer_id: 'cus_1', dimension: 'api_calls', quantity: 42 }),
            verb: 'post',
            path: '/v1/usage/events',
            body: { subscription_id: 'sub_1', customer_id: 'cus_1', dimension: 'api_calls', quantity: 42 },
        },
    ],

    creditNotes: [
        { method: 'create', call: (r) => r.creditNotes.create(body), verb: 'post', path: '/v1/credit-notes', body },
        { method: 'list', call: (r) => r.creditNotes.list(listParams), verb: 'get', path: '/v1/credit-notes', params: listParams },
    ],

    quotes: [
        { method: 'create', call: (r) => r.quotes.create(body), verb: 'post', path: '/v1/quotes', body },
        { method: 'list', call: (r) => r.quotes.list(listParams), verb: 'get', path: '/v1/quotes', params: listParams },
        { method: 'get', call: (r) => r.quotes.get('qt_1'), verb: 'get', path: '/v1/quotes/qt_1' },
        { method: 'update', call: (r) => r.quotes.update('qt_1', body), verb: 'put', path: '/v1/quotes/qt_1', body },
        { method: 'send', call: (r) => r.quotes.send('qt_1'), verb: 'post', path: '/v1/quotes/qt_1/send' },
        { method: 'accept', call: (r) => r.quotes.accept('qt_1'), verb: 'post', path: '/v1/quotes/qt_1/accept' },
        { method: 'decline', call: (r) => r.quotes.decline('qt_1'), verb: 'post', path: '/v1/quotes/qt_1/decline' },
        { method: 'convert', call: (r) => r.quotes.convert('qt_1'), verb: 'post', path: '/v1/quotes/qt_1/convert' },
        { method: 'delete', call: (r) => r.quotes.delete('qt_1'), verb: 'delete', path: '/v1/quotes/qt_1' },
    ],

    webhooks: [
        {
            method: 'create',
            call: (r) => r.webhooks.create({ url: 'https://example.com/hook', event_types: ['invoice.paid'] }),
            verb: 'post',
            path: '/v1/webhooks',
            body: { url: 'https://example.com/hook', event_types: ['invoice.paid'] },
        },
        { method: 'list', call: (r) => r.webhooks.list(), verb: 'get', path: '/v1/webhooks' },
        { method: 'delete', call: (r) => r.webhooks.delete('wh_1'), verb: 'delete', path: '/v1/webhooks/wh_1' },
    ],

    events: [
        { method: 'list', call: (r) => r.events.list(listParams), verb: 'get', path: '/v1/events', params: listParams },
        { method: 'types', call: (r) => r.events.types(), verb: 'get', path: '/v1/events/types' },
    ],

    mandates: [
        { method: 'create', call: (r) => r.mandates.create(body), verb: 'post', path: '/v1/mandates', body },
        { method: 'list', call: (r) => r.mandates.list(listParams), verb: 'get', path: '/v1/mandates', params: listParams },
        { method: 'get', call: (r) => r.mandates.get('mnd_1'), verb: 'get', path: '/v1/mandates/mnd_1' },
        { method: 'revoke', call: (r) => r.mandates.revoke('mnd_1'), verb: 'post', path: '/v1/mandates/mnd_1/revoke' },
    ],

    gifts: [
        { method: 'purchase', call: (r) => r.gifts.purchase(body), verb: 'post', path: '/v1/gifts/purchase', body },
        {
            method: 'redeem',
            call: (r) => r.gifts.redeem({ code: 'GIFT-1234' }),
            verb: 'post',
            path: '/v1/gifts/redeem',
            body: { code: 'GIFT-1234' },
        },
        { method: 'list', call: (r) => r.gifts.list(listParams), verb: 'get', path: '/v1/gifts', params: listParams },
    ],

    referrals: [
        { method: 'create', call: (r) => r.referrals.create(body), verb: 'post', path: '/v1/referrals', body },
        { method: 'list', call: (r) => r.referrals.list(listParams), verb: 'get', path: '/v1/referrals', params: listParams },
        {
            method: 'generateCode',
            call: (r) => r.referrals.generateCode({ customer_id: 'cus_1' }),
            verb: 'post',
            path: '/v1/referrals/generate-code',
            body: { customer_id: 'cus_1' },
        },
        { method: 'qualify', call: (r) => r.referrals.qualify('ref_1'), verb: 'post', path: '/v1/referrals/ref_1/qualify' },
    ],

    entitlements: [
        {
            method: 'setForPlan',
            call: (r) =>
                r.entitlements.setForPlan('plan_1', [
                    { feature_key: 'sso', kind: 'boolean', bool_value: true },
                    { feature_key: 'seats', kind: 'limit', limit_value: 25 },
                ]),
            verb: 'put',
            path: '/v1/plans/plan_1/entitlements',
            body: [
                { feature_key: 'sso', kind: 'boolean', bool_value: true },
                { feature_key: 'seats', kind: 'limit', limit_value: 25 },
            ],
        },
        {
            method: 'getForPlan',
            call: (r) => r.entitlements.getForPlan('plan_1'),
            verb: 'get',
            path: '/v1/plans/plan_1/entitlements',
        },
        {
            method: 'forCustomer',
            call: (r) => r.entitlements.forCustomer('cus_1'),
            verb: 'get',
            path: '/v1/customers/cus_1/entitlements',
        },
        {
            method: 'check',
            call: (r) => r.entitlements.check('cus_1', 'sso'),
            verb: 'get',
            path: '/v1/entitlements/check',
            params: { customer_id: 'cus_1', feature: 'sso' },
        },
    ],

    ledger: [
        { method: 'accounts', call: (r) => r.ledger.accounts(), verb: 'get', path: '/v1/ledger/accounts' },
        {
            method: 'entries',
            call: (r) => r.ledger.entries({ account_id: 'acct_1' }),
            verb: 'get',
            path: '/v1/ledger/entries',
            params: { account_id: 'acct_1' },
        },
    ],
};

describe('Recurso constructor', () => {
    it('creates one axios client with Bearer auth and JSON content type', () => {
        expect(mocks.create).toHaveBeenCalledTimes(1);
        expect(mocks.create).toHaveBeenCalledWith({
            baseURL: BASE_URL,
            headers: {
                Authorization: `Bearer ${API_KEY}`,
                'Content-Type': 'application/json',
            },
        });
    });

    it('defaults baseURL to http://localhost:8080', () => {
        mocks.create.mockClear();
        new Recurso(API_KEY);
        expect(mocks.create).toHaveBeenCalledWith(
            expect.objectContaining({ baseURL: 'http://localhost:8080' }),
        );
    });
});

for (const [resource, methods] of Object.entries(cases)) {
    describe(resource, () => {
        for (const c of methods) {
            it(`${c.method} -> ${c.verb.toUpperCase()} ${c.path}`, async () => {
                const result = await c.call(recurso);

                // Response envelope is unwrapped to `.data`.
                expect(result).toBe(PAYLOAD);

                // Only the expected verb fired, exactly once.
                const fn = mocks.client[c.verb];
                expect(fn).toHaveBeenCalledTimes(1);
                for (const other of ['get', 'post', 'put', 'delete'] as const) {
                    if (other !== c.verb) expect(mocks.client[other]).not.toHaveBeenCalled();
                }

                // Verb-specific argument shape.
                if (c.verb === 'get') {
                    expect(fn).toHaveBeenCalledWith(c.path, { params: c.params });
                } else if (c.verb === 'delete') {
                    expect(fn).toHaveBeenCalledWith(c.path);
                } else {
                    expect(fn).toHaveBeenCalledWith(c.path, c.body);
                }
            });
        }
    });
}

describe('list params passthrough', () => {
    it('forwards params untouched (no cloning, filtering, or renaming)', async () => {
        const params = { page: 1, limit: 10, q: 'acme', status: 'active', custom_filter: 'x' };
        await recurso.customers.list(params);
        // Same object reference: proves untouched passthrough.
        expect(mocks.client.get.mock.calls[0][1].params).toBe(params);
    });

    it('sends undefined params when a list method is called without arguments', async () => {
        await recurso.plans.list();
        expect(mocks.client.get).toHaveBeenCalledWith('/v1/plans', { params: undefined });
    });
});

describe('customers.create country default', () => {
    it('defaults country to US when omitted', async () => {
        await recurso.customers.create({ email: 'a@b.co', name: 'A' });
        expect(mocks.client.post).toHaveBeenCalledWith('/v1/customers', {
            country: 'US',
            email: 'a@b.co',
            name: 'A',
        });
    });

    it('caller-supplied country wins over the default', async () => {
        await recurso.customers.create({ email: 'a@b.co', name: 'A', country: 'DE' });
        expect(mocks.client.post.mock.calls[0][1].country).toBe('DE');
    });
});

describe('plans.create interval_count default', () => {
    it('defaults interval_count to 1 when omitted', async () => {
        await recurso.plans.create({
            name: 'Pro',
            code: 'PRO',
            amount: 2900,
            currency: 'USD',
            interval_unit: 'month',
        });
        expect(mocks.client.post).toHaveBeenCalledWith('/v1/plans', {
            interval_count: 1,
            name: 'Pro',
            code: 'PRO',
            amount: 2900,
            currency: 'USD',
            interval_unit: 'month',
        });
    });

    it('caller-supplied interval_count wins over the default', async () => {
        await recurso.plans.create({
            name: 'Quarterly',
            code: 'Q',
            amount: 900,
            currency: 'USD',
            interval_unit: 'month',
            interval_count: 3,
        });
        expect(mocks.client.post.mock.calls[0][1].interval_count).toBe(3);
    });
});

describe('invoices.pdfUrl', () => {
    it('builds the public PDF URL from the client baseURL without an HTTP call', () => {
        const url = recurso.invoices.pdfUrl('inv_42');
        expect(url).toBe(`${BASE_URL}/v1/invoices/inv_42/pdf`);
        for (const verb of ['get', 'post', 'put', 'delete'] as const) {
            expect(mocks.client[verb]).not.toHaveBeenCalled();
        }
    });
});

describe('error propagation', () => {
    it('rejects with the underlying axios error', async () => {
        const boom = Object.assign(new Error('Request failed with status code 422'), {
            response: { status: 422, data: { error: 'validation failed' } },
        });
        mocks.client.post.mockRejectedValueOnce(boom);
        await expect(recurso.subscriptions.create({ customer_id: 'c', plan_id: 'p' })).rejects.toBe(boom);
    });
});

describe('API surface completeness', () => {
    it('the test table covers every public resource method on the SDK', () => {
        const sdkSurface: Record<string, string[]> = {};
        for (const [key, value] of Object.entries(recurso as unknown as Record<string, unknown>)) {
            if (key === 'client') continue; // private axios instance
            if (value && typeof value === 'object' && !Array.isArray(value)) {
                const methodNames = Object.entries(value)
                    .filter(([, v]) => typeof v === 'function')
                    .map(([name]) => name)
                    .sort();
                if (methodNames.length > 0) sdkSurface[key] = methodNames;
            }
        }

        const tested: Record<string, string[]> = {};
        for (const [resource, methods] of Object.entries(cases)) {
            tested[resource] = methods.map((m) => m.method).sort();
        }
        // pdfUrl is covered by its own dedicated (non-HTTP) test.
        tested.invoices = [...tested.invoices, 'pdfUrl'].sort();

        expect(tested).toEqual(sdkSurface);
    });
});

// Sanity: the real axios module is fully mocked in this suite.
describe('mocking', () => {
    it('axios.create is the mocked factory', () => {
        expect(axios.create).toBe(mocks.create);
    });
});
