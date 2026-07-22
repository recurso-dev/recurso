import { useQuery } from "@tanstack/react-query";
import {
  Bell,
  UserPlus,
  CreditCard,
  AlertCircle,
  UserRound,
} from "lucide-react";

import { endpoints } from "@/lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Card } from "@/components/ui/card";

export default function Notifications() {
  const {
    data: notifications = [],
    isLoading: loading,
    error: queryError,
  } = useQuery({
    queryKey: ["notifications"],
    queryFn: async () => {
      const response = await endpoints.getEvents({ limit: 20 });
      // Map events to notification format.
      return (response.data.data || []).map((evt) => {
        let title = "System Event";
        let description = `Event ${evt.type} occurred on ${evt.object_type}`;
        let Icon = Bell;

        switch (evt.type) {
          case "subscription.created":
            title = "New Subscription";
            description = "A new subscription was created.";
            Icon = UserPlus;
            break;
          case "invoice.paid":
            title = "Payment Received";
            description = "Invoice was successfully paid.";
            Icon = CreditCard;
            break;
          case "invoice.payment_failed":
            title = "Payment Failed";
            description = "Payment processing failed for invoice.";
            Icon = AlertCircle;
            break;
          case "customer.created":
            title = "New Customer";
            description = "A new customer has been registered.";
            Icon = UserRound;
            break;
          default:
            title = evt.type.replace(".", " ").replace(/\b\w/g, (l) => l.toUpperCase());
            Icon = Bell;
        }

        return {
          id: evt.id,
          title,
          description,
          time: new Date(evt.created_at).toLocaleString(),
          Icon,
          read: false, // No backend support yet.
        };
      });
    },
  });
  const error = queryError ? "Failed to load notifications." : null;

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="Notifications"
        description="Stay updated with the latest events."
      />

      {loading ? (
        <div className="py-10 text-center text-sm text-muted-foreground">
          Loading notifications...
        </div>
      ) : error ? (
        <Card>
          <EmptyState
            icon={AlertCircle}
            title={error}
            description="Please refresh to try again."
          />
        </Card>
      ) : notifications.length === 0 ? (
        <Card>
          <EmptyState
            icon={Bell}
            title="No notifications found."
            description="Events will appear here as activity happens in your account."
          />
        </Card>
      ) : (
        <div className="flex flex-col gap-3">
          {notifications.map((note) => {
            const Icon = note.Icon;
            return (
              <Card key={note.id} className="flex items-start gap-4 p-4">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-emerald-700">
                  <Icon className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <h3 className="text-sm font-semibold text-foreground">{note.title}</h3>
                    <span className="shrink-0 text-xs text-muted-foreground">{note.time}</span>
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">{note.description}</p>
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
