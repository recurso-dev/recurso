import { Link } from "react-router";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// CustomerName renders a resolved customer name as a link to the customer's
// detail (/customers/:id), falling back to a mono short id while (or if) the
// lookup hasn't resolved. Pass link={false} where the cell already lives
// inside another interactive element (e.g. the first cell of an onRowClick
// table, which DataTable wraps in a <button>).
export function CustomerName({ id, names, link = true }) {
  if (id && names[id]) {
    if (link) {
      return (
        <Link
          to={`/customers/${id}`}
          onClick={(e) => e.stopPropagation()}
          className="text-sm text-foreground underline-offset-2 hover:text-primary hover:underline"
        >
          {names[id]}
        </Link>
      );
    }
    return <span className="text-sm text-foreground">{names[id]}</span>;
  }
  return (
    <span className="font-mono text-xs text-muted-foreground">
      {id ? `${String(id).slice(0, 8)}…` : "—"}
    </span>
  );
}

// CustomerSelect is the standard customer picker for create dialogs —
// replaces raw "paste a UUID" inputs with the same name (email) dropdown
// the full-page create flows use.
export function CustomerSelect({ id, value, onChange, customers, placeholder = "Select a customer" }) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger id={id}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {customers.map((c) => (
          <SelectItem key={c.id} value={c.id}>
            {c.name} {c.email ? `(${c.email})` : ""}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
