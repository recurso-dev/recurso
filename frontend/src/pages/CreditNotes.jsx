import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Plus, Receipt } from "lucide-react";

import { endpoints } from "../lib/api";
import CreditNoteDetail from "../components/slide-overs/CreditNoteDetail";
import { formatCurrency, formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

const creditStatusVariant = (status) =>
  ({
    issued: "info",
    used: "success",
    active: "success",
    pending_approval: "warning",
    rejected: "destructive",
  })[status] || "neutral";

const CreditNotes = () => {
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [selectedNote, setSelectedNote] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  const {
    data: creditNotes = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["credit-notes"],
    queryFn: async () => (await endpoints.getCreditNotes()).data.data || [],
  });
  const error = queryError ? "Failed to load credit notes." : null;

  const filteredNotes = creditNotes.filter(
    (cn) =>
      cn.id.toLowerCase().includes(search.toLowerCase()) ||
      (cn.customer?.name || "").toLowerCase().includes(search.toLowerCase())
  );

  const handleRowClick = (note) => {
    setSelectedNote(note);
    setIsDetailOpen(true);
  };

  const closeDetail = () => {
    setIsDetailOpen(false);
    setTimeout(() => setSelectedNote(null), 300);
  };

  const columns = [
    {
      key: "id",
      header: "ID",
      cell: (cn) => (
        <span className="font-mono text-xs text-foreground">
          {cn.reference || cn.id.slice(0, 8)}
        </span>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      cell: (cn) => (
        <span className="text-muted-foreground">
          {cn.customer ? cn.customer.name : "Unknown Customer"}
        </span>
      ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (cn) => (
        <span className="tabular-nums font-medium text-foreground">
          {formatCurrency(cn.amount, cn.currency)}
        </span>
      ),
    },
    {
      key: "balance",
      header: "Balance",
      align: "right",
      cell: (cn) => (
        <span className="tabular-nums text-muted-foreground">
          {formatCurrency(cn.balance, cn.currency)}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (cn) => (
        <Badge variant={creditStatusVariant(cn.status)} className="capitalize">
          {(cn.status || "").replace("_", " ")}
        </Badge>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (cn) => (
        <span className="text-muted-foreground">{formatDate(cn.created_at)}</span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Credit Notes"
        description="Manage customer credits and refunds."
        actions={
          <Button onClick={() => navigate("/credit-notes/new")}>
            <Plus className="h-4 w-4" />
            Create credit note
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={filteredNotes}
        loading={loading}
        error={error}
        onRetry={refetch}
        onRowClick={handleRowClick}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search by ID or customer...",
        }}
        empty={{
          icon: Receipt,
          title: search ? "No matching credit notes" : "No credit notes yet",
          description: search
            ? "Try adjusting your search."
            : "Issue a credit note to get started.",
          action: !search ? (
            <Button onClick={() => navigate("/credit-notes/new")}>
              <Plus className="h-4 w-4" />
              Create credit note
            </Button>
          ) : null,
        }}
      />

      <CreditNoteDetail
        creditNote={selectedNote}
        isOpen={isDetailOpen}
        onClose={closeDetail}
      />
    </div>
  );
};

export default CreditNotes;
