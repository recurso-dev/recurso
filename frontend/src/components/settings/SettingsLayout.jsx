import { useMemo } from "react";
import { NavLink, Outlet } from "react-router";
import { useQuery } from "@tanstack/react-query";
import {
  SlidersHorizontal,
  ShieldCheck,
  UserCog,
  Receipt,
  FileCheck2,
  MapPinned,
  Globe,
  Bot,
  Building2,
  ArrowDownToLine,
  CreditCard,
  Palette,
} from "lucide-react";

import { endpoints } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";

const EU = new Set([
  "AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR", "DE", "GR",
  "HU", "IE", "IT", "LV", "LT", "LU", "MT", "NL", "PL", "PT", "RO", "SK",
  "SI", "ES", "SE",
]);
// Mirrors the backend's RegimeForCountry so the nav floats + badges the tax
// setup that matches the business country.
const regionOf = (cc) => {
  const c = (cc || "").toUpperCase();
  if (c === "IN") return "IN";
  if (c === "US") return "US";
  if (c === "GB" || EU.has(c)) return "EU";
  return "other";
};

// Every settings destination, in nav order. `end` marks the index route so it
// isn't kept active on child paths. Security lives at /security but belongs in
// this nav conceptually.
const SECTIONS = [
  { to: "/settings", end: true, group: "General", icon: SlidersHorizontal, title: "General" },
  { to: "/settings/invoice-branding", group: "Billing documents", icon: Palette, title: "Invoice branding" },
  { to: "/settings/entities", group: "Billing documents", icon: Building2, title: "Legal entities" },
  { to: "/settings/gst", group: "Taxes & compliance", icon: Receipt, title: "GST configuration", regions: ["IN"] },
  { to: "/settings/irp", group: "Taxes & compliance", icon: FileCheck2, title: "E-invoicing (IRP)", regions: ["IN"] },
  { to: "/settings/eu-einvoice", group: "Taxes & compliance", icon: Globe, title: "EU e-invoicing", regions: ["EU"] },
  { to: "/settings/tax-nexus", group: "Taxes & compliance", icon: MapPinned, title: "US sales-tax nexus", regions: ["US"] },
  { to: "/settings/tax-us", group: "Taxes & compliance", icon: Receipt, title: "US tax identity (W-9)", regions: ["US"] },
  { to: "/security", group: "Account & platform", icon: ShieldCheck, title: "Security" },
  { to: "/team", group: "Account & platform", icon: UserCog, title: "Team" },
  { to: "/settings/mcp", group: "Account & platform", icon: Bot, title: "MCP server" },
  { to: "/settings/import", group: "Account & platform", icon: ArrowDownToLine, title: "Import data" },
  { to: "/settings/billing", group: "Account & platform", icon: CreditCard, title: "Billing & plan" },
];

const GROUPS = ["General", "Billing documents", "Taxes & compliance", "Account & platform"];

export default function SettingsLayout() {
  // The primary entity's country drives which tax setups are relevant.
  const { data: entities = [] } = useQuery({
    queryKey: ["entities"],
    queryFn: async () => (await endpoints.getEntities()).data.data || [],
  });
  const primary = entities.find((e) => e.is_primary) || entities[0] || null;
  const region = regionOf(primary?.country_code || "");

  const grouped = useMemo(() => {
    const relevant = (l) => (region !== "other" && l.regions?.includes(region) ? 0 : 1);
    return GROUPS.map((group) => ({
      group,
      links: SECTIONS.map((l, i) => ({ l, i }))
        .filter(({ l }) => l.group === group)
        .sort((a, b) => relevant(a.l) - relevant(b.l) || a.i - b.i)
        .map(({ l }) => l),
    }));
  }, [region]);

  return (
    <div className="flex flex-col gap-8 lg:flex-row">
      <nav aria-label="Settings sections" className="lg:w-56 lg:shrink-0">
        <div className="space-y-5 lg:sticky lg:top-6">
          {grouped.map(({ group, links }) =>
            links.length ? (
              <div key={group}>
                {group !== "General" && (
                  <h2 className="mb-1 px-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {group}
                  </h2>
                )}
                <ul className="space-y-0.5">
                  {links.map((l) => {
                    const isRelevant = region !== "other" && l.regions?.includes(region);
                    return (
                      <li key={l.to}>
                        <NavLink
                          to={l.to}
                          end={l.end}
                          className={({ isActive }) =>
                            cn(
                              "flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors",
                              isActive
                                ? "bg-muted font-medium text-foreground"
                                : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
                            )
                          }
                        >
                          <l.icon className="h-4 w-4 shrink-0" />
                          <span className="flex-1 truncate">{l.title}</span>
                          {isRelevant && (
                            <Badge variant="success" className="text-[10px]">
                              For your region
                            </Badge>
                          )}
                        </NavLink>
                      </li>
                    );
                  })}
                </ul>
              </div>
            ) : null
          )}
        </div>
      </nav>

      <div className="min-w-0 flex-1">
        <Outlet />
      </div>
    </div>
  );
}
