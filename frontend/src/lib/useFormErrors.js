import { useCallback, useState } from "react";

/**
 * useFormErrors — the shared submit-validation contract (UX_RULES / T4):
 * an errors object keyed by control id, plus focus-first-error on a failed
 * submit so keyboard and screen-reader users land on the problem. FormField
 * renders each message as a role="alert" live region and wires
 * aria-invalid/aria-describedby; this hook standardizes the other half so
 * every form validates the same way (no toast-only or native-only paths).
 *
 * Usage:
 *   const { errors, validate, clearError } = useFormErrors();
 *   const onSubmit = (e) => {
 *     e.preventDefault();
 *     const errs = {};
 *     if (!name.trim()) errs.name = "Name is required.";
 *     if (!validate(errs)) return; // stores messages + focuses first bad field
 *     // ...submit
 *   };
 *   <FormField htmlFor="name" error={errors.name}>
 *     <Input id="name" onChange={(e) => { setName(e.target.value); clearError("name"); }} />
 *   </FormField>
 */
export function useFormErrors() {
  const [errors, setErrors] = useState({});

  // validate(errs) stores the messages and returns whether the form is clean.
  // Keys must be control ids so the first offender can receive focus (focus
  // scrolls the control into view natively — no motion assumptions).
  const validate = useCallback((errs) => {
    setErrors(errs);
    const first = Object.keys(errs)[0];
    if (first) {
      requestAnimationFrame(() => document.getElementById(first)?.focus());
      return false;
    }
    return true;
  }, []);

  // clearError(key) drops one field's message — call it from the field's
  // onChange so the alert disappears as soon as the user starts fixing it.
  const clearError = useCallback((key) => {
    setErrors((prev) => {
      if (!(key in prev)) return prev;
      const next = { ...prev };
      delete next[key];
      return next;
    });
  }, []);

  return { errors, validate, clearError, setErrors };
}
