import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// CustomerName renders a resolved customer name, falling back to a mono
// short id while (or if) the lookup hasn't resolved.
export function CustomerName({ id, names }) {
  if (id && names[id]) {
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
export function CustomerSelect({ value, onChange, customers, placeholder = "Select a customer" }) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger>
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
