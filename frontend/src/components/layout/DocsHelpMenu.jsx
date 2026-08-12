import { Link, useLocation } from "react-router";
import { HelpCircle, BookOpen, Library, Code2, Sparkles, ExternalLink } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { docsUrlFor, DOCS_GUIDES, DOCS_API_REFERENCE } from "@/lib/docsLinks";

// Small external-link row used for docs.recurso.dev destinations.
function DocsLink({ href, icon: Icon, children, hint }) {
  return (
    <DropdownMenuItem asChild>
      <a href={href} target="_blank" rel="noopener noreferrer" className="cursor-pointer">
        <Icon className="text-muted-foreground" />
        <span className="flex-1">{children}</span>
        {hint ? (
          <span className="text-[11px] text-subtle">{hint}</span>
        ) : (
          <ExternalLink className="h-3.5 w-3.5 text-subtle" />
        )}
      </a>
    </DropdownMenuItem>
  );
}

/**
 * DocsHelpMenu — the top-bar "?" affordance. Deep-links to the guide for the
 * page you're currently on, plus the guide index, API reference and Ask AI.
 */
export function DocsHelpMenu({ pageTitle }) {
  const { pathname } = useLocation();
  const contextualUrl = docsUrlFor(pathname);
  const contextualLabel = pageTitle && pageTitle !== "Recurso" ? `${pageTitle} guide` : "Guide for this page";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-white text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        aria-label="Help and documentation"
      >
        <HelpCircle className="h-4 w-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-60">
        <DropdownMenuLabel>Help &amp; docs</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DocsLink href={contextualUrl} icon={BookOpen}>
          {contextualLabel}
        </DocsLink>
        <DocsLink href={DOCS_GUIDES} icon={Library}>
          Browse all guides
        </DocsLink>
        <DocsLink href={DOCS_API_REFERENCE} icon={Code2}>
          API reference
        </DocsLink>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <Link to="/ask" className="cursor-pointer">
            <Sparkles className="text-success" />
            <span className="flex-1">Ask AI</span>
          </Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default DocsHelpMenu;
