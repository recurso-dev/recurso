import { Link, useLocation } from "react-router";
import { SearchX } from "lucide-react";

import { Button } from "@/components/ui/button";

/**
 * NotFound — a real 404. The old wildcard silently redirected to Home, which
 * hid broken links from everyone (the Collections → /customers/:id dead link
 * shipped unnoticed because of it; DASHBOARD_UI_AUDIT §7).
 */
export default function NotFound() {
  const { pathname } = useLocation();
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center px-6 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-border bg-muted">
        <SearchX className="h-5 w-5 text-subtle" aria-hidden="true" />
      </div>
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">
        Page not found
      </h1>
      <p className="mt-2 max-w-md text-sm text-muted-foreground">
        There's no page at{" "}
        <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">{pathname}</code>.
        If a link brought you here, it's broken — worth reporting.
      </p>
      <Button asChild className="mt-6">
        <Link to="/">Go to Home</Link>
      </Button>
    </div>
  );
}
