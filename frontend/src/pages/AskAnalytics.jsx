import { useState } from "react";
import { Sparkles, Send } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const EXAMPLES = [
  "What was my MRR growth over the last 3 months?",
  "Which plan has the most active subscriptions?",
  "How many invoices are overdue, and for how much?",
];

// Render an arbitrary array-of-objects result as a table; scalars as text.
function ResultView({ data }) {
  if (data == null) return null;
  if (Array.isArray(data) && data.length > 0 && typeof data[0] === "object" && data[0] !== null) {
    const cols = Object.keys(data[0]);
    return (
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c} className="whitespace-nowrap font-mono text-xs">
                  {c}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((row, i) => (
              <TableRow key={i}>
                {cols.map((c) => (
                  <TableCell key={c} className="whitespace-nowrap text-sm">
                    {row[c] == null ? "—" : String(row[c])}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    );
  }
  return (
    <pre className="overflow-x-auto rounded-md bg-muted p-4 font-mono text-xs text-foreground">
      {typeof data === "string" ? data : JSON.stringify(data, null, 2)}
    </pre>
  );
}

// Natural-language analytics: the backend translates the question to a
// tenant-scoped SQL query and returns the rows plus the query it ran.
const AskAnalytics = () => {
  const [question, setQuestion] = useState("");
  const [asking, setAsking] = useState(false);
  const [result, setResult] = useState(null); // { question, data, query }
  const [error, setError] = useState(null);

  const ask = async (q) => {
    const text = (q ?? question).trim();
    if (!text) return;
    setAsking(true);
    setError(null);
    setResult(null);
    try {
      const res = await api.askAnalytics(text);
      setResult({ question: text, data: res.data.data, query: res.data.query });
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

  return (
    <div>
      <PageHeader
        title="Ask your data"
        description="Ask billing questions in plain language — answered from your own tenant's data."
      />

      <form
        onSubmit={(e) => {
          e.preventDefault();
          ask();
        }}
        className="flex gap-2"
      >
        <Input
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="e.g. Which customers churned last month?"
          aria-label="Question"
        />
        <Button type="submit" disabled={asking || !question.trim()}>
          <Send className="h-4 w-4" />
          {asking ? "Thinking…" : "Ask"}
        </Button>
      </form>

      <div className="mt-3 flex flex-wrap gap-2">
        {EXAMPLES.map((ex) => (
          <button
            key={ex}
            type="button"
            onClick={() => {
              setQuestion(ex);
              ask(ex);
            }}
            className="rounded-full border border-border px-3 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            {ex}
          </button>
        ))}
      </div>

      <div className="mt-8">
        {error ? (
          <p className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-800">{error}</p>
        ) : result ? (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{result.question}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <ResultView data={result.data} />
              {result.query && (
                <details className="text-xs text-muted-foreground">
                  <summary className="cursor-pointer select-none">Show the SQL it ran</summary>
                  <pre className="mt-2 overflow-x-auto rounded-md bg-muted p-3 font-mono">
                    {result.query}
                  </pre>
                </details>
              )}
            </CardContent>
          </Card>
        ) : (
          !asking && (
            <EmptyState
              icon={Sparkles}
              title="Ask anything about your billing data"
              description="Questions run as read-only, tenant-scoped queries. Try one of the examples above."
            />
          )
        )}
      </div>
    </div>
  );
};

export default AskAnalytics;
