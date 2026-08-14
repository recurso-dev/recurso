import { Link } from "react-router";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { MotionStagger } from "@/components/patterns/MotionReveal";

const humanizeEvent = (type = "") =>
  type.replace(/[._]/g, " ").replace(/^./, (c) => c.toUpperCase());

/**
 * ObjectTimeline — this object's slice of the business-event stream
 * (GET /events?object_id=), for object-page rails. The counterpart of
 * AuditTrail: the audit trail is who changed the configuration; the
 * timeline is what happened to the object (created, paid, renewed…).
 *
 * Props:
 *  - objectId: the object's uuid
 *  - limit:    max entries shown (default 8)
 */
export function ObjectTimeline({ objectId, limit = 8 }) {
  const { data: events, isLoading, error } = useQuery({
    queryKey: ["objectTimeline", objectId],
    queryFn: async () =>
      (await endpoints.getEvents({ object_id: objectId, limit })).data.data || [],
    enabled: Boolean(objectId),
  });

  if (isLoading) {
    return (
      <div className="space-y-3" aria-busy="true">
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-2/3" />
        <Skeleton className="h-4 w-1/2" />
      </div>
    );
  }
  if (error) {
    return (
      <p className="text-sm text-muted-foreground" role="status">
        Couldn&apos;t load the timeline.
      </p>
    );
  }
  if (!events?.length) {
    return (
      <p className="text-sm text-muted-foreground">No events recorded for this object yet.</p>
    );
  }

  return (
    <div>
      <ol className="space-y-3">
        <MotionStagger step={45}>
        {events.map((ev, i) => (
          <li key={ev.id} className="relative flex gap-3">
            {/* Rail dot + connecting line */}
            <span className="flex flex-col items-center" aria-hidden="true">
              <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary/60" />
              {i < events.length - 1 && <span className="mt-1 w-px flex-1 bg-border" />}
            </span>
            <div className="min-w-0 pb-1">
              <p className="truncate text-sm font-medium text-foreground">
                {humanizeEvent(ev.type)}
              </p>
              <p className="text-xs text-muted-foreground">{formatDateTime(ev.created_at)}</p>
            </div>
          </li>
        ))}
        </MotionStagger>
      </ol>
      <Link
        to="/events"
        className="mt-4 inline-block text-xs font-medium text-primary underline-offset-2 hover:underline"
      >
        View all events
      </Link>
    </div>
  );
}
