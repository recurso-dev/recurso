package recurso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Event is a billing event emitted for the tenant.
type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	ObjectType string          `json:"object_type"`
	ObjectID   string          `json:"object_id"`
	Data       json.RawMessage `json:"data"`
	CreatedAt  time.Time       `json:"created_at"`
}

// RedeliverResult is returned by Redeliver.
type RedeliverResult struct {
	EventID          string `json:"event_id"`
	DeliveriesQueued int    `json:"deliveries_queued"`
}

// EventListParams paginates the event feed.
type EventListParams struct {
	Limit  int
	Offset int
}

// EventsService groups the event-feed endpoints.
type EventsService struct{ client *Client }

// List returns the tenant's event feed, newest first.
func (s *EventsService) List(ctx context.Context, params *EventListParams) ([]Event, error) {
	path := "/events"
	if params != nil {
		q := newQuery().int("limit", params.Limit)
		if params.Offset != 0 {
			q.int("offset", params.Offset)
		}
		path = q.apply(path)
	}
	return getData[[]Event](ctx, s.client, http.MethodGet, path, nil)
}

// Types returns all subscribable event types.
func (s *EventsService) Types(ctx context.Context) ([]string, error) {
	return getData[[]string](ctx, s.client, http.MethodGet, "/events/types", nil)
}

// Deliveries lists delivery attempts of an event across all endpoints.
func (s *EventsService) Deliveries(ctx context.Context, id string) ([]EventDelivery, error) {
	return getData[[]EventDelivery](ctx, s.client, http.MethodGet, fmt.Sprintf("/events/%s/deliveries", id), nil)
}

// Redeliver re-enqueues delivery of an event to all subscribed endpoints.
func (s *EventsService) Redeliver(ctx context.Context, id string) (*RedeliverResult, error) {
	out, err := getData[RedeliverResult](ctx, s.client, http.MethodPost, fmt.Sprintf("/events/%s/redeliver", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
