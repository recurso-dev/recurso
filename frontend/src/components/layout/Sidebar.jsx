import { NavLink } from "react-router";
import { Layers, BookOpen, ArrowUpRight } from "lucide-react";

import { cn } from "@/lib/utils";
import { NAV_GROUPS } from "@/lib/navigation";
import { DOCS_HOME } from "@/lib/docsLinks";

function SidebarItem({ to, label, icon: Icon, end, onNavigate }) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onNavigate}
      className={({ isActive }) =>
        cn(
          "group flex items-center gap-2.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
          isActive
            ? "bg-primary/10 text-primary"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        )
      }
    >
      {({ isActive }) => (
        <>
          <Icon
            className={cn(
              "h-4 w-4 shrink-0",
              isActive ? "text-primary" : "text-subtle group-hover:text-foreground"
            )}
          />
          <span className="truncate">{label}</span>
        </>
      )}
    </NavLink>
  );
}

/**
 * NavList — the grouped destination list, shared verbatim between the
 * desktop sidebar and the mobile drawer so the two can never diverge.
 * `onNavigate` lets the drawer close itself when a destination is chosen.
 */
export function NavList({ onNavigate }) {
  return (
    <nav aria-label="Primary" className="flex-1 overflow-y-auto px-3 py-4">
      {NAV_GROUPS.map((group, i) => (
        <div key={group.label || i} className="mb-5 last:mb-0">
          {group.label && (
            <p className="mb-1.5 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {group.label}
            </p>
          )}
          <div className="space-y-0.5">
            {group.items.map((item) => (
              <SidebarItem key={item.to} {...item} onNavigate={onNavigate} />
            ))}
          </div>
        </div>
      ))}
    </nav>
  );
}

export function Brand() {
  return (
    <div className="flex h-16 shrink-0 items-center gap-2.5 border-b border-border px-5">
      <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
        <Layers className="h-4 w-4" />
      </div>
      <span className="text-sm font-semibold tracking-tight text-foreground">
        Recurso
      </span>
    </div>
  );
}

export function DocsFooterLink() {
  return (
    <div className="shrink-0 border-t border-border p-3">
      <a
        href={DOCS_HOME}
        target="_blank"
        rel="noopener noreferrer"
        className="group flex items-center gap-2.5 rounded-md px-3 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <BookOpen className="h-4 w-4 shrink-0 text-subtle group-hover:text-foreground" />
        <span className="flex-1 truncate">Documentation</span>
        <ArrowUpRight className="h-3.5 w-3.5 text-subtle/60 group-hover:text-subtle" />
      </a>
    </div>
  );
}

/**
 * Desktop sidebar. Hidden below lg — the mobile drawer (DashboardLayout)
 * renders the same NavList behind a hamburger instead, so the 240px rail
 * never consumes a phone viewport (audit R1).
 */
export function Sidebar() {
  return (
    <aside className="hidden h-full w-60 flex-col border-r border-border bg-background lg:flex">
      <Brand />
      <NavList />
      <DocsFooterLink />
    </aside>
  );
}

export default Sidebar;
