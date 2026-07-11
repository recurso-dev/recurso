# Spec: Proration UX

## Objective
Build the dashboard UI to support plan-change prorations. The backend (`SubscriptionService.UpdateSubscription`) already calculates prorations and handles the billing logic. The UI needs to allow users to select a new plan, preview the proration charges/credits, and confirm the change.

## Tech Stack
- React (Vite)
- Tailwind CSS (or existing UI library used in the dashboard)
- Recurso Go Backend (for the API)

## Commands
Build: `cd frontend && npm run build`
Test: `cd frontend && npm test`
Lint: `cd frontend && npm run lint`
Dev: `cd frontend && npm run dev`

## Project Structure
```
frontend/
  src/
    pages/
      Subscriptions/
        ChangePlanModal.tsx    → The modal UI for upgrading/downgrading
    components/
      ProrationPreview.tsx     → Component to display the line-item preview
    hooks/
      useProrationPreview.ts   → Hook to fetch the preview data from the API
```

## Code Style
```tsx
// ProrationPreview.tsx
export function ProrationPreview({ currentPlan, newPlan, previewData }: Props) {
  if (!previewData) return <LoadingSpinner />;
  
  return (
    <div className="border rounded p-4">
      <h3 className="font-semibold">Proration Summary</h3>
      <div className="flex justify-between">
        <span>Unused time on {currentPlan.name}</span>
        <span className="text-green-600">- {formatCurrency(previewData.creditAmount)}</span>
      </div>
      <div className="flex justify-between">
        <span>Remaining time on {newPlan.name}</span>
        <span>+ {formatCurrency(previewData.chargeAmount)}</span>
      </div>
      <div className="border-t mt-2 pt-2 flex justify-between font-bold">
        <span>Amount Due Today</span>
        <span>{formatCurrency(previewData.netAmount)}</span>
      </div>
    </div>
  );
}
```

## Testing Strategy
- **Unit Tests**: Test the `ProrationPreview` component with various mocks (negative net amount, positive net amount, zero net amount).
- **Integration Tests**: Mock the API response for the preview endpoint to ensure the UI updates correctly without making real API calls during testing.

## Boundaries
- **Always**: Show a clear breakdown of credits vs. charges so the user understands exactly what they are paying for today.
- **Ask first**: Before creating a new API endpoint for the preview if the existing `UpdateSubscription` endpoint can be modified to support a `?dry_run=true` parameter.
- **Never**: Allow the user to submit the plan change without explicitly clicking a "Confirm & Pay" (or "Confirm Change") button after the preview is loaded.

## Success Criteria
- [ ] Users can click "Change Plan" on a subscription in the dashboard.
- [ ] A modal displays the available plans.
- [ ] Selecting a plan displays a real-time preview of the proration calculation.
- [ ] Confirming the change updates the subscription via the API.

## Open Questions
- Does the backend currently have a "dry run" endpoint to fetch the proration preview without actually mutating the subscription? If not, we need to build that backend piece as well.
