import { useEffect, useMemo, useState } from "react";
import { AreaChart, BarChart } from "@tremor/react";
import {
  Sparkles,
  Send,
  Download,
  RotateCw,
  Trash2,
  Table2,
  BarChart3,
  Code2,
} from "lucide-react";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  makeChartTooltip,
  chartCategoryColors,
  chartDefaults,
} from "@/components/charts/ChartTooltip";
import { docsUrlFor } from "@/lib/docsLinks";

const EXAMPLES = [
  "What was my MRR growth over the last 3 months?",
  "Which plan has the most active subscriptions?",
  "How many invoices are overdue, and for how much?",
  "Top 10 customers by revenue",
];

// First-run example gallery, grouped to teach the breadth of what's askable —
// revenue, customers, collections, and the books themselves.
const EXAMPLE_GROUPS = [
  {
    label: "Revenue",
    items: [
      "What was my MRR growth over the last 3 months?",
      "Revenue by plan this quarter",
    ],
  },
  {
    label: "Customers",
    items: [
      "Top 10 customers by revenue",
      "Which customers churned last month?",
    ],
  },
  {
    label: "Collections",
    items: [
      "How many invoices are overdue, and for how much?",
      "Which payments failed this week?",
    ],
  },
  {
    label: "Books",
    items: [
      "What is my deferred revenue balance right now?",
      "Credit notes issued this month, with amounts",
    ],
  },
];

const HISTORY_KEY = "recurso.ask.history.v1";
const HISTORY_CAP = 25;

// ---- helpers --------------------------------------------------------------

// Group integers/decimals for readability; leave strings alone.
function formatCell(v) {
  if (v == null) return "—";
  if (typeof v === "number") {
    return Number.isInteger(v)
      ? v.toLocaleString()
      : v.toLocaleString(undefined, { maximumFractionDigits: 2 });
  }
  if (typeof v === "boolean") return v ? "Yes" : "No";
  return String(v);
}

const isNumericCol = (rows, col) =>
  rows.every((r) => r[col] == null || typeof r[col] === "number");

// Decide whether a result is worth charting and how. Returns null when a table
// is the honest representation (too many rows, no clear label+measure shape).
function inferChart(rows) {
  if (!Array.isArray(rows) || rows.length < 2 || rows.length > 60) return null;
  const first = rows[0];
  if (!first || typeof first !== "object") return null;
  const cols = Object.keys(first);
  const numeric = cols.filter((c) => isNumericCol(rows, c));
  const labels = cols.filter((c) => !numeric.includes(c));
  if (labels.length === 0 || numeric.length === 0) return null;

  const index = labels[0];
  // Distinct-ish labels only — charting a column that's the same value every
  // row (or all unique ids) is noise.
  const distinct = new Set(rows.map((r) => String(r[index]))).size;
  if (distinct < 2) return null;

  const categories = numeric.slice(0, 4);
  const timeLike =
    /date|month|day|period|week|year|time|quarter/i.test(index) ||
    rows.every((r) => typeof r[index] === "string" && /^\d{4}-\d{2}/.test(r[index]));
  return { type: timeLike ? "area" : "bar", index, categories };
}

function toCSV(rows) {
  if (!Array.isArray(rows) || rows.length === 0) return "";
  const cols = Object.keys(rows[0]);
  const esc = (v) =>
    v == null
      ? ""
      : /[",\n]/.test(String(v))
        ? `"${String(v).replace(/"/g, '""')}"`
        : String(v);
  return [cols.join(","), ...rows.map((r) => cols.map((c) => esc(r[c])).join(","))].join("\n");
}

function downloadCSV(rows, name) {
  const blob = new Blob([toCSV(rows)], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${name}.csv`;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 5_000);
}

const chartNumberFmt = (v) => (typeof v === "number" ? v.toLocaleString() : v);

// ---- result rendering -----------------------------------------------------

function ResultChart({ rows, spec }) {
  const tooltip = useMemo(() => makeChartTooltip(chartNumberFmt), []);
  const common = {
    data: rows,
    index: spec.index,
    categories: spec.categories,
    colors: chartCategoryColors,
    customTooltip: tooltip,
    valueFormatter: chartNumberFmt,
    showLegend: spec.categories.length > 1,
    yAxisWidth: 56,
    className: "h-64",
    ...chartDefaults,
  };
  return spec.type === "area" ? (
    <AreaChart {...common} showGradient />
  ) : (
    <BarChart {...common} />
  );
}

function ResultTable({ rows }) {
  const cols = Object.keys(rows[0]);
  return (
    <div className="max-h-80 overflow-auto rounded-md border border-border">
      <Table>
        <TableHeader className="sticky top-0 bg-muted/60 backdrop-blur">
          <TableRow>
            {cols.map((c) => (
              <TableHead
                key={c}
                className={`whitespace-nowrap font-mono text-xs ${
                  isNumericCol(rows, c) ? "text-right" : ""
                }`}
              >
                {c}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, i) => (
            <TableRow key={i}>
              {cols.map((c) => (
                <TableCell
                  key={c}
                  className={`whitespace-nowrap text-sm ${
                    isNumericCol(rows, c) ? "text-right tabular-nums" : ""
                  }`}
                >
                  {formatCell(row[c])}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function ResultBody({ data }) {
  const isTabular =
    Array.isArray(data) &&
    data.length > 0 &&
    typeof data[0] === "object" &&
    data[0] !== null;

  const chart = useMemo(() => (isTabular ? inferChart(data) : null), [data, isTabular]);
  const [view, setView] = useState(chart ? "chart" : "table");
  useEffect(() => setView(chart ? "chart" : "table"), [chart]);

  if (data == null || (Array.isArray(data) && data.length === 0)) {
    return <p className="text-sm text-muted-foreground">No rows matched that question.</p>;
  }

  if (!isTabular) {
    return (
      <pre className="overflow-x-auto rounded-md bg-muted p-4 font-mono text-xs text-foreground">
        {typeof data === "string" ? data : JSON.stringify(data, null, 2)}
      </pre>
    );
  }

  return (
    <div className="space-y-3">
      {chart && (
        <div className="flex items-center gap-1">
          <Button
            variant={view === "chart" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setView("chart")}
          >
            <BarChart3 className="mr-1.5 h-3.5 w-3.5" />
            Chart
          </Button>
          <Button
            variant={view === "table" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setView("table")}
          >
            <Table2 className="mr-1.5 h-3.5 w-3.5" />
            Table
          </Button>
        </div>
      )}
      {view === "chart" && chart ? (
        <ResultChart rows={data} spec={chart} />
      ) : (
        <ResultTable rows={data} />
      )}
    </div>
  );
}

function ResultCard({ entry, onRerun, onRemove }) {
  const [showSql, setShowSql] = useState(false);
  const rowCount = Array.isArray(entry.data) ? entry.data.length : null;
  return (
    <Card>
      <CardContent className="space-y-4 p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="flex items-center gap-2 text-sm font-semibold text-foreground">
              <Sparkles className="h-4 w-4 shrink-0 text-emerald-600" />
              <span className="truncate">{entry.question}</span>
            </p>
            {rowCount != null && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                {rowCount} {rowCount === 1 ? "row" : "rows"}
              </p>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <Button variant="ghost" size="sm" onClick={() => onRerun(entry.question)} title="Ask again">
              <RotateCw className="h-3.5 w-3.5" />
            </Button>
            {Array.isArray(entry.data) && entry.data.length > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => downloadCSV(entry.data, "recurso-answer")}
                title="Download CSV"
              >
                <Download className="h-3.5 w-3.5" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              className="text-stone-400 hover:text-red-600"
              onClick={() => onRemove(entry.id)}
              title="Remove"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>

        <ResultBody data={entry.data} />

        {entry.query && (
          <div>
            <button
              type="button"
              onClick={() => setShowSql((s) => !s)}
              className="flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
              <Code2 className="h-3.5 w-3.5" />
              {showSql ? "Hide SQL" : "Show the SQL it ran"}
            </button>
            {showSql && (
              <pre className="mt-2 overflow-x-auto rounded-md bg-muted p-3 font-mono text-xs">
                {entry.query}
              </pre>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ---- persistent history ---------------------------------------------------

function loadHistory() {
  try {
    const raw = localStorage.getItem(HISTORY_KEY);
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

// ---- page -----------------------------------------------------------------

const AskAnalytics = () => {
  const [question, setQuestion] = useState("");
  const [asking, setAsking] = useState(false);
  const [error, setError] = useState(null);
  const [history, setHistory] = useState(loadHistory);

  // Persist the thread so a reload (or coming back later) keeps the answers.
  useEffect(() => {
    try {
      localStorage.setItem(HISTORY_KEY, JSON.stringify(history));
    } catch {
      /* quota / privacy mode — history just won't persist */
    }
  }, [history]);

  const ask = async (q) => {
    const text = (q ?? question).trim();
    if (!text || asking) return;
    setAsking(true);
    setError(null);
    try {
      const res = await api.askAnalytics(text);
      const entry = {
        id: `${Date.now()}-${Math.round(Math.random() * 1e6)}`,
        question: text,
        data: res.data.data,
        query: res.data.query,
        ts: Date.now(),
      };
      setHistory((h) => [entry, ...h].slice(0, HISTORY_CAP));
      setQuestion("");
    } catch (err) {
      setError(
        err?.response?.status === 503
          ? "GenAI analytics isn't configured on this deployment — set OPENAI_API_KEY on the server to enable it."
          : err?.response?.data?.error?.message ||
              "Could not answer that — try rephrasing the question."
      );
    } finally {
      setAsking(false);
    }
  };

  const removeEntry = (id) => setHistory((h) => h.filter((e) => e.id !== id));
  const clearAll = () => setHistory([]);

  // First run = nothing asked yet: the page becomes an invitation (hero input
  // + example gallery). Once there's history, it compacts into a working tool.
  const isFirstRun = history.length === 0 && !asking && !error;

  return (
    <div>
      <PageHeader
        title="Ask your data"
        description="Ask billing questions in plain language — answered from your own tenant's data as read-only, tenant-scoped queries."
        actions={
          history.length > 0 ? (
            <Button variant="outline" size="sm" onClick={clearAll}>
              <Trash2 className="mr-1.5 h-4 w-4" />
              Clear history
            </Button>
          ) : null
        }
      />

      {/* The input is the star of this page: large, focused, always first. */}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          ask();
        }}
        className={isFirstRun ? "mx-auto mt-10 flex w-full max-w-2xl gap-2" : "flex gap-2"}
      >
        <div className="relative flex-1">
          <Sparkles className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-emerald-500" aria-hidden />
          <Input
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            placeholder="Ask anything about your billing data…"
            aria-label="Question"
            autoFocus
            className={
              isFirstRun
                ? "h-12 rounded-xl pl-10 text-base shadow-sm"
                : "h-10 rounded-lg pl-10"
            }
          />
        </div>
        <Button
          type="submit"
          disabled={asking || !question.trim()}
          className={isFirstRun ? "h-12 rounded-xl px-5" : ""}
        >
          <Send className="h-4 w-4" />
          {asking ? "Thinking…" : "Ask"}
        </Button>
      </form>

      {isFirstRun ? (
        /* First run: a gallery that teaches the breadth of what's askable. */
        <div className="mx-auto mt-8 grid w-full max-w-2xl grid-cols-1 gap-4 sm:grid-cols-2">
          {EXAMPLE_GROUPS.map((group) => (
            <div key={group.label} className="rounded-xl border border-border p-4">
              <p className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                {group.label}
              </p>
              <div className="space-y-1">
                {group.items.map((ex) => (
                  <button
                    key={ex}
                    type="button"
                    disabled={asking}
                    onClick={() => ask(ex)}
                    className="block w-full rounded-md px-2 py-1.5 text-left text-sm text-foreground/80 transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50"
                  >
                    {ex}
                  </button>
                ))}
              </div>
            </div>
          ))}
          <p className="col-span-full text-center text-xs text-muted-foreground">
            Read-only, tenant-scoped queries — and every answer shows the SQL it
            ran.{" "}
            <a
              href={docsUrlFor("/ask")}
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-foreground"
            >
              How it works
            </a>
          </p>
        </div>
      ) : (
        <div className="mt-3 flex flex-wrap gap-2">
          {EXAMPLES.map((ex) => (
            <button
              key={ex}
              type="button"
              disabled={asking}
              onClick={() => ask(ex)}
              className="rounded-full border border-border px-3 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50"
            >
              {ex}
            </button>
          ))}
        </div>
      )}

      {error && (
        <p className="mt-4 rounded-md bg-red-50 px-4 py-3 text-sm text-red-800" role="alert">
          {error}
        </p>
      )}

      <div className="mt-6 space-y-4">
        {asking && (
          <Card aria-busy="true" aria-label="Working on your question">
            <CardContent className="space-y-3 p-5">
              <div className="flex items-center gap-3 text-sm text-muted-foreground">
                <Sparkles className="h-4 w-4 animate-pulse text-emerald-600" />
                Translating your question into a tenant-scoped query…
              </div>
              <div className="space-y-2">
                <div className="h-3 w-3/4 animate-pulse rounded bg-muted" />
                <div className="h-3 w-1/2 animate-pulse rounded bg-muted" />
                <div className="h-3 w-2/3 animate-pulse rounded bg-muted" />
              </div>
            </CardContent>
          </Card>
        )}

        {history.length > 0 && (
          <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            Recent answers
          </p>
        )}
        {history.map((entry) => (
          <ResultCard
            key={entry.id}
            entry={entry}
            onRerun={ask}
            onRemove={removeEntry}
          />
        ))}
      </div>
    </div>
  );
};

export default AskAnalytics;
