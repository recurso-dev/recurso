export declare class Recurso {
    private client;
    constructor(apiKey: string, baseURL?: string);
    customers: {
        create: (data: {
            email: string;
            name: string;
            line1?: string;
            city?: string;
            state?: string;
            zip?: string;
            country?: string;
        }) => Promise<any>;
    };
    plans: {
        create: (data: {
            name: string;
            code: string;
            amount: number;
            currency: string;
            interval_unit: "month" | "year";
            interval_count?: number;
        }) => Promise<any>;
    };
    subscriptions: {
        create: (data: {
            customer_id: string;
            plan_id: string;
            coupon_code?: string;
        }) => Promise<any>;
    };
    coupons: {
        create: (data: {
            code: string;
            discount_type: "percent" | "amount";
            discount_value: number;
            duration: "forever" | "once";
        }) => Promise<any>;
    };
}
