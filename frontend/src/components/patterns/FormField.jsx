import * as React from "react";

import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";

// Native tags a <label for> may target. A composite child (Input, Select,
// Textarea, …) is assumed to forward `id` to one of these.
const LABELLABLE_TAGS = new Set(["input", "select", "textarea", "button", "meter", "output", "progress"]);

/**
 * FormField — wraps a control with a label, optional description, and error.
 * Pass the actual control (Input / Select / textarea) as children.
 *
 * Accessibility: the label is associated with the control without the caller
 * having to invent ids. The control's id is, in order, the caller's `htmlFor`
 * (caller-owned — the child is never touched), the child's own `id`, or a
 * generated `useId()` handed to a single labellable child (a native control or
 * a component that forwards `id` to one; a wrapper <div> is left alone). The
 * description and error get ids derived from it and are wired onto the child
 * via `aria-describedby`, and an errored control receives `aria-invalid` — so a
 * screen reader announces the helper text and the validation message against
 * the field. The error is a `role="alert"` live region. Any `aria-*` the caller
 * already set is preserved and merged.
 *
 * Props:
 *  - label:       string
 *  - htmlFor:     string (id of the control; optional — generated when omitted)
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
  const autoId = React.useId();
  // Only a single real element (not a Fragment/array) can take aria-* props or
  // an id; a multi-child FormField is left untouched.
  const single = React.isValidElement(children) && children.type !== React.Fragment;
  const labellable =
    single && (typeof children.type !== "string" || LABELLABLE_TAGS.has(children.type));
  const childId = single ? children.props.id : undefined;
  // A generated id is only injected when the caller gave neither an htmlFor nor
  // a child id — with htmlFor the control (possibly nested in a wrapper) is the
  // caller's to id, exactly as before.
  const injectId = htmlFor == null && childId == null && labellable;
  const controlId = htmlFor ?? childId ?? (injectId ? autoId : undefined);
  const idBase = controlId ?? autoId;
  const descId = description ? `${idBase}-description` : undefined;
  const errId = error ? `${idBase}-error` : undefined;
  const describedBy = [descId, errId].filter(Boolean).join(" ") || undefined;

  // Thread the generated id plus aria-describedby (+ aria-invalid on error)
  // onto the child control, merging with anything the caller already set.
  let control = children;
  if (single && (injectId || describedBy || error)) {
    const merged = [children.props["aria-describedby"], describedBy]
      .filter(Boolean)
      .join(" ");
    control = React.cloneElement(children, {
      ...(injectId ? { id: controlId } : {}),
      "aria-invalid": error ? true : children.props["aria-invalid"],
      "aria-describedby": merged || undefined,
    });
  }

  return (
    <div className={cn("space-y-1.5", className)}>
      {label && (
        <Label htmlFor={controlId} className="text-foreground">
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
        <p
          id={errId}
          role="alert"
          className="animate-motion-reveal text-xs font-medium text-destructive"
        >
          {error}
        </p>
      )}
    </div>
  );
}

export default FormField;
