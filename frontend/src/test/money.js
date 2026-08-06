/**
 * money(text) — a React Testing Library text matcher for the <Money> component.
 *
 * <Money> (components/ui/money.jsx) renders each Intl currency part in its own
 * span (the symbol in a muted `.money-symbol`), so an amount like "$49.00" is
 * split across multiple nodes and `getByText("$49.00")` never matches. This
 * matcher targets the outer `.money` element whose full textContent equals the
 * expected string.
 *
 * Usage:
 *   import { money } from "@/test/money";
 *   expect(screen.getByText(money("$49.00"))).toBeInTheDocument();
 *   expect(screen.getAllByText(money("₹5,000.00")).length).toBeGreaterThan(0);
 */
export function money(text) {
  return (_content, el) =>
    Boolean(el?.classList?.contains("money")) && el.textContent === text;
}

export default money;
