import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";

/**
 * FormField — wraps a control with a label, optional description, and error.
 * Pass the actual control (Input / Select / textarea) as children.
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
  return (
    <div className={cn("space-y-1.5", className)}>
      {label && (
        <Label htmlFor={htmlFor} className="text-foreground">
          {label}
          {required && <span className="ml-0.5 text-red-500">*</span>}
        </Label>
      )}
      {description && (
        <p className="text-xs text-muted-foreground">{description}</p>
      )}
      {children}
      {error && <p className="text-xs font-medium text-red-600">{error}</p>}
    </div>
  );
}

export default FormField;
