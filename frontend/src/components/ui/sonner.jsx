import { Toaster as Sonner, toast } from "sonner";

/**
 * Toaster — global toast host. Mount once near the app root.
 * Styled to match the light enterprise theme (white surface, zinc border,
 * emerald success). Use the `toast` export to fire toasts from anywhere:
 *   import { toast } from "@/components/ui/sonner"
 *   toast.success("Customer created")
 */
const Toaster = (props) => {
  return (
    <Sonner
      theme="light"
      position="bottom-right"
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-white group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg group-[.toaster]:rounded-lg",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
          success: "group-[.toaster]:text-emerald-700",
          error: "group-[.toaster]:text-red-700",
        },
      }}
      {...props}
    />
  );
};

export { Toaster, toast };
