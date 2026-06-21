"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Recurso = void 0;
const axios_1 = __importDefault(require("axios"));
class Recurso {
    constructor(apiKey, baseURL = 'http://localhost:8080') {
        this.customers = {
            create: async (data) => {
                // Default to US if not provided to satisfy backend "len=2" requirement for now
                const payload = { country: 'US', ...data };
                const res = await this.client.post('/v1/customers', payload);
                return res.data;
            },
        };
        this.plans = {
            create: async (data) => {
                const res = await this.client.post('/v1/plans', { interval_count: 1, ...data });
                return res.data;
            },
        };
        this.subscriptions = {
            create: async (data) => {
                const res = await this.client.post('/v1/subscriptions', data);
                return res.data;
            },
        };
        this.coupons = {
            create: async (data) => {
                const res = await this.client.post('/v1/coupons', data);
                return res.data;
            }
        };
        this.client = axios_1.default.create({
            baseURL,
            headers: {
                Authorization: `Bearer ${apiKey}`,
                'Content-Type': 'application/json',
            },
        });
    }
}
exports.Recurso = Recurso;
