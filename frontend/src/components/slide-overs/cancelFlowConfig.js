// Shared, non-component constants/helpers for the cancel-flow step editor.
// Kept in a plain module so the editor component file stays fast-refresh clean.

export const OFFER_TYPES = [
  { value: "discount", label: "Discount" },
  { value: "pause", label: "Pause subscription" },
  { value: "plan_switch", label: "Switch plan" },
  { value: "trial_extension", label: "Extend trial" },
  { value: "custom", label: "Custom" },
];

// Default config objects seeded per step type, mirroring the backend's default flow.
export const defaultConfigFor = (type) =>
  ({
    survey: { questions: ["Too expensive", "Missing features", "Other"], allow_feedback: true },
    offer: {
      headline: "Before you go, we'd like to offer you a deal",
      offers: [{ type: "discount", discount_percent: 20, discount_duration_months: 3 }],
    },
    confirmation: {
      message: "Are you sure you want to cancel?",
      confirm_button: "Yes, cancel my subscription",
      cancel_button: "No, keep my subscription",
    },
  })[type] || {};
