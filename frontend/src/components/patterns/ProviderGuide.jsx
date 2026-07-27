import { ExternalLink } from "lucide-react";

/**
 * ProviderGuide — "where to find this credential" steps shown inside a
 * connect sheet, so nobody has to leave the flow to google a provider's
 * console path. Pass the provider's `guide` ({ steps, url, urlLabel }).
 */
export function ProviderGuide({ guide }) {
  if (!guide?.steps?.length) return null;
  return (
    <div className="rounded-md border border-border bg-muted/40 p-3 text-xs">
      <p className="mb-1.5 font-medium text-foreground">Where to find this</p>
      <ol className="ml-4 list-decimal space-y-1 text-muted-foreground">
        {guide.steps.map((step) => (
          <li key={step}>{step}</li>
        ))}
      </ol>
      {guide.url && (
        <a
          href={guide.url}
          target="_blank"
          rel="noreferrer"
          className="mt-2 inline-flex items-center gap-1 font-medium text-primary hover:underline"
        >
          {guide.urlLabel || "Open provider console"}
          <ExternalLink className="h-3 w-3" />
        </a>
      )}
    </div>
  );
}

export default ProviderGuide;
