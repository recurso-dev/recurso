import { Money } from "@/components/ui/money";
import { CopyableId } from "@/components/ui/copyable-id";
import { formatDate, formatDateTime } from "@/lib/utils";

/**
 * Cell primitives (DASHBOARD_REDESIGN.md Stage 4) — the one way to render
 * money, dates, and identifiers in table cells. Using these makes drift
 * impossible: money can't be left-aligned, dates can't invent formats, IDs
 * can't lose their mono/copy treatment.
 *
 * Pair with the column config: MoneyCell implies align:"right" on the column.
 */

export function MoneyCell({ amountMinor, currency }) {
  if (amountMinor == null) return <span className="text-muted-foreground">—</span>;
  return <Money amountMinor={amountMinor} currency={currency} />;
}

export function DateCell({ value, time = false }) {
  return (
    <span className="whitespace-nowrap text-muted-foreground">
      {time ? formatDateTime(value) : formatDate(value)}
    </span>
  );
}

export function IdCell({ value, label = "ID" }) {
  return <CopyableId value={value} label={label} />;
}
