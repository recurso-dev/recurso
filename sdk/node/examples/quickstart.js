const { Recurso } = require('../dist/index.js');
const axios = require('axios');

// Helper to register a tenant on the fly for the demo
async function getApiKey() {
  try {
    const res = await axios.post('http://localhost:8080/auth/register', {
      name: 'SDK User',
      email: `sdk-${Date.now()}@example.com`
    });
    return res.data.api_key;
  } catch (e) {
    console.error("Registration failed", e.response?.data);
    process.exit(1);
  }
}

async function main() {
  console.log("🚀 Starting Recurso SDK Quickstart...");
  
  // 1. Get a fresh API Key
  const apiKey = await getApiKey();
  console.log(`🔑 Obtained API Key: ${apiKey}`);

  // 2. Initialize SDK
  const recurso = new Recurso(apiKey);

  // 3. Create Plan
  console.log("📦 Creating Plan...");
  const plan = await recurso.plans.create({
    name: "Pro Plan SDK",
    code: "pro-sdk",
    amount: 5000, // $50.00
    currency: "USD",
    interval_unit: "month"
  });
  console.log(`   ✅ Plan created: ${plan.id}`);

  // 4. Create Customer
  console.log("👤 Creating Customer...");
  const customer = await recurso.customers.create({
    name: "Alice Developer",
    email: "alice@dev.com",
    country: "US"
  });
  console.log(`   ✅ Customer created: ${customer.id}`);

  // 5. Create Subscription
  console.log("✨ Creating Subscription...");
  const sub = await recurso.subscriptions.create({
    customer_id: customer.id,
    plan_id: plan.id
  });
  console.log(`   ✅ Subscription created: ${sub.id}`);
  
  console.log("\n🎉 Quickstart Complete!");
}

main();
