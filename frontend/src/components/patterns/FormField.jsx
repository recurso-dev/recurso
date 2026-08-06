import * as React from "react";

import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";

/**
 * FormField — wraps a control with a label, optional description, and error.
 * Pass the actual control (Input / Select / textarea) as children.
 *
 * Accessibility: when `htmlFor` is set, the description and error are given ids
 * and wired onto a single child control via `aria-describedby`, and an errored
 * control receives `aria-invalid` — so a screen reader announces the helper text
 * and the validation message against the field. The error is a `role="alert"`
 * live region. Any `aria-*` the caller already set is preserved and merged.
 *
 * Props:
 *  - label:       string
 *  - htmlFor:     string (id of the control, for a11y)
 *  - required:    boolean (renders a red asterisk)
 *  - description: string (muted helper under the label)
 *  - error:       string (validation message, renders red)
 *  - children:    the control
 *
 * Example:
 *   <FormField label="Email" htmlFor="email" required error={errors.email}>
 *     <Input id="email" value={email} onChange={...} />
 *   </FormField>
 */
export function FormField({
  label,
  htmlFor,
  required,
  description,
  error,
  children,
  className,
}) {
  const descId = description && htmlFor ? `${htmlFor}-description` : undefined;
  const errId = error && htmlFor ? `${htmlFor}-error` : undefined;
  const describedBy = [descId, errId].filter(Boolean).join(" ") || undefined;

  // Thread aria-describedby (+ aria-invalid on error) onto a single child
  // control, merging with anything the caller already set. Fragments/arrays
  // aren't cloneable, so a multi-child FormField is left untouched.
  let control = children;
  if (React.isValidElement(children) && (describedBy || error)) {
    const merged = [children.props["aria-describedby"], describedBy]
      .filter(Boolean)
      .join(" ");
    control = React.cloneElement(children, {
      "aria-invalid": error ? true : children.props["aria-invalid"],
      "aria-describedby": merged || undefined,
    });
  }

  return (
    <div className={cn("space-y-1.5", className)}>
      {label && (
        <Label htmlFor={htmlFor} className="text-foreground">
          {label}
          {required && <span className="ml-0.5 text-destructive">*</span>}
        </Label>
      )}
      {description && (
        <p id={descId} className="text-xs text-muted-foreground">
          {description}
        </p>
      )}
      {control}
      {error && (
        <p id={errId} role="alert" className="text-xs font-medium text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}

export default FormField;
