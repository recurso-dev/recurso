import axios, { AxiosInstance } from 'axios';

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

    public customers = {
        create: async (data: {
            email: string;
            name: string;
            line1?: string;
            city?: string;
            state?: string;
            zip?: string;
            country?: string;
        }) => {
            // Default to US if not provided to satisfy backend "len=2" requirement for now
            const payload = { country: 'US', ...data };
            const res = await this.client.post('/v1/customers', payload);
            return res.data;
        },
    };

    public plans = {
        create: async (data: {
            name: string;
            code: string;
            amount: number;
            currency: string;
            interval_unit: 'month' | 'year';
            interval_count?: number;
        }) => {
            const res = await this.client.post('/v1/plans', { interval_count: 1, ...data });
            return res.data;
        },
    };

    public subscriptions = {
        create: async (data: { customer_id: string; plan_id: string; coupon_code?: string }) => {
            const res = await this.client.post('/v1/subscriptions', data);
            return res.data;
        },
    };

    public coupons = {
        create: async (data: { code: string; discount_type: 'percent' | 'amount'; discount_value: number; duration: 'forever' | 'once' }) => {
            const res = await this.client.post('/v1/coupons', data);
            return res.data;
        }
    }
}
