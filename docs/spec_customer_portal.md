# Spec: Customer Portal Enhancements (Payment & Disputes)

## Objective
Add self-service flows to the customer portal (the hosted page where end-customers manage their billing). Specifically, customers need the ability to update their default payment method and initiate a dispute or query on an invoice.

## Tech Stack
- React (Vite)
- Stripe Elements / Razorpay Checkout (for secure card capture)
- Recurso Go Backend

## Commands
Build: `cd frontend && npm run build`
Test: `cd frontend && npm test`
Lint: `cd frontend && npm run lint`
Dev: `cd frontend && npm run dev`

## Project Structure
```
frontend/
  src/
    portal/
      pages/
        PaymentMethods.tsx     → UI to view and add/remove cards
        InvoiceDetail.tsx      → Add "Dispute" button
      components/
        AddCardModal.tsx       → Stripe/Razorpay integration
internal/
  adapter/
    handler/
      portal_api.go            → Add endpoints for adding cards and disputing
```

## Code Style
```tsx
// AddCardModal.tsx (Stripe Example)
const handleSubmit = async (event) => {
  event.preventDefault();
  if (!stripe || !elements) return;

  setIsLoading(true);
  const { setupIntent, error } = await stripe.confirmCardSetup(clientSecret, {
    payment_method: {
      card: elements.getElement(CardElement),
    }
  });

  if (error) {
    setErrorMsg(error.message);
  } else {
    await attachMethodToCustomer(setupIntent.payment_method);
    onSuccess();
  }
  setIsLoading(false);
};
```

## Testing Strategy
- **Frontend Tests**: Ensure the "Add Card" button correctly mounts the Stripe/Razorpay elements (mocking the external scripts).
- **Backend Tests**: Validate that the portal API correctly associates the new payment method with the correct customer ID and tenant ID (preventing IDOR vulnerabilities).

## Boundaries
- **Always**: Use Stripe SetupIntents (or Razorpay equivalent) to collect card details securely without the server touching raw PANs.
- **Ask first**: Before building a complex internal ticketing system for disputes. The MVP should likely just trigger an email notification to the tenant admin and flag the invoice in the DB.
- **Never**: Render the portal without verifying the secure, time-bound access token.

## Success Criteria
- [ ] End-customers can log into the portal, view their current default card, and replace it with a new one securely.
- [ ] End-customers can click "Dispute" on a specific invoice, enter a reason, and submit it.
- [ ] The tenant admin receives an email notification when a dispute is submitted.

## Open Questions
- For the dispute flow, should we just email the tenant admin, or should we also add a `status="disputed"` flag to the Invoice model to pause automated dunning/collections?
