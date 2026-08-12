import { MoneyCell } from "./cells";

/**
 * moneyColumn — column-config helper that bundles the right-alignment rule
 * with the cell so callers can't forget it:
 *   moneyColumn({ key: "total", header: "Amount",
 *                 amount: (r) => r.total, currency: (r) => r.currency })
 */
export function moneyColumn({ key, header, amount, currency, ...rest }) {
  return {
    key,
    header,
    align: "right",
    sortValue: (row) => amount(row),
    cell: (row) => <MoneyCell amountMinor={amount(row)} currency={currency(row)} />,
    ...rest,
  };
}
