import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Package } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { useDebounce } from "../hooks/useDebounce";
import { cn, formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

const STATUS_FILTERS = ["all", "active", "archived"];
const PAGE_SIZE = 10;

export default function Products() {
  const navigate = useNavigate();

  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebounce(search, 500);

  const fetchProducts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = { q: debouncedSearch, limit: PAGE_SIZE, page };
      const response = await api.getPlans(params);
      // The backend's catalog unit is the Plan; this page presents plans as
      // "products" with their price points as variants. Mapping preserved.
      const plans = response.data.data || [];
      const transformed = plans.map((p) => ({
        id: p.id,
        name: p.name,
        description: p.description || "No description",
        status: p.active ? "active" : "archived",
        prices: p.prices ? p.prices.length : 0,
        created_at: p.created_at,
      }));
      setProducts(transformed);
    } catch (err) {
      setError(
        err?.response?.data?.error?.message || err?.message || "Failed to load products"
      );
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, page]);

  useEffect(() => {
    fetchProducts();
  }, [fetchProducts]);

  // Reset to page 1 whenever the search query changes.
  useEffect(() => {
    setPage(1);
  }, [debouncedSearch]);

  // Client-side status filter over the fetched page (preserved).
  const filteredProducts = useMemo(
    () =>
      products.filter((p) => statusFilter === "all" || p.status === statusFilter),
    [products, statusFilter]
  );

  const hasFilters = search || statusFilter !== "all";

  const columns = [
    {
      key: "name",
      header: "Product name",
      cell: (p) => <span className="font-medium text-foreground">{p.name}</span>,
    },
    {
      key: "description",
      header: "Description",
      cell: (p) => (
        <span className="block max-w-xs truncate text-muted-foreground">
          {p.description}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (p) => (
        <Badge variant={p.status === "active" ? "success" : "neutral"}>
          {p.status.charAt(0).toUpperCase() + p.status.slice(1)}
        </Badge>
      ),
    },
    {
      key: "prices",
      header: "Prices",
      cell: (p) => <span className="tabular-nums text-muted-foreground">{p.prices}</span>,
    },
    {
      key: "created",
      header: "Created",
      cell: (p) => (
        <span className="text-muted-foreground">{formatDate(p.created_at)}</span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Product Catalog"
        description="Browse the catalog of plans and their price points."
        actions={
          <Button onClick={() => navigate("/plans/new")}>
            <Plus className="h-4 w-4" />
            Create product
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={filteredProducts}
        loading={loading}
        error={error}
        onRetry={fetchProducts}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search products...",
        }}
        toolbar={
          <div className="flex items-center gap-1 rounded-lg border border-border bg-white p-0.5">
            {STATUS_FILTERS.map((f) => (
              <button
                key={f}
                onClick={() => setStatusFilter(f)}
                className={cn(
                  "rounded-md px-3 py-1 text-sm font-medium capitalize transition-colors",
                  statusFilter === f
                    ? "bg-emerald-50 text-emerald-700"
                    : "text-zinc-500 hover:text-zinc-900"
                )}
              >
                {f}
              </button>
            ))}
          </div>
        }
        empty={{
          icon: Package,
          title: hasFilters ? "No matching products" : "No products yet",
          description: hasFilters
            ? "Try adjusting your search or filters."
            : "Create a plan to populate your product catalog.",
          action: !hasFilters ? (
            <Button onClick={() => navigate("/plans/new")}>
              <Plus className="h-4 w-4" />
              Create product
            </Button>
          ) : null,
        }}
        pagination={{
          page,
          onPrev: () => setPage((p) => Math.max(1, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: products.length >= PAGE_SIZE,
        }}
      />
    </div>
  );
}
