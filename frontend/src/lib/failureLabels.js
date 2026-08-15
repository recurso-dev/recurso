// Humanize the raw gateway / ACH failure codes stored on an invoice's last
// payment attempt, so an operator reads "Insufficient funds" instead of
// "insufficient_funds". Shared by Collections and the Home attention surface.
// Unknown codes fall back to a title-cased version of the code itself.
export const FAILURE_LABELS = {
  card_declined: "Card declined",
  insufficient_funds: "Insufficient funds",
  expired_card: "Expired card",
  incorrect_cvc: "Incorrect CVC",
  processing_error: "Processing error",
  do_not_honor: "Do not honor",
  lost_card: "Lost card",
  stolen_card: "Stolen card",
  authentication_required: "Authentication required",
  ach_return: "ACH return",
  R01: "ACH: insufficient funds",
  R02: "ACH: account closed",
  R03: "ACH: no account",
  R08: "ACH: payment stopped",
  R10: "ACH: unauthorized",
};

export const humanizeFailure = (code) => {
  if (!code) return "—";
  if (FAILURE_LABELS[code]) return FAILURE_LABELS[code];
  return code.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
};
