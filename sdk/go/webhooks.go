package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// WebhookEndpoint is a registered webhook delivery endpoint. Secret is only
// populated in the creation response.
type WebhookEndpoint struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret"`
	Events    []string  `json:"events"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// EventDelivery is a single delivery attempt of an event to an endpoint.
type EventDelivery struct {
	ID                string     `json:"id"`
	EventID           string     `json:"event_id"`
	WebhookEndpointID string     `json:"webhook_endpoint_id"`
	EndpointURL       string     `json:"endpoint_url"`
	Status            string     `json:"status"`
	Attempts          int        `json:"attempts"`
	LastStatusCode    int        `json:"last_status_code"`
	LastError         string     `json:"last_error"`
	NextRetryAt       *time.Time `json:"next_retry_at"`
	DeliveredAt       *time.Time `json:"delivered_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

// WebhookCreateParams registers a webhook endpoint.
type WebhookCreateParams struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// DeliveryListParams filters an endpoint's deliveries.
type DeliveryListParams struct {
	Limit  int
	Offset int
	Status string
}

// WebhooksService groups the webhook-endpoint endpoints.
type WebhooksService struct{ client *Client }

// Create registers a webhook endpoint. The signing secret is returned only here.
func (s *WebhooksService) Create(ctx context.Context, params *WebhookCreateParams) (*WebhookEndpoint, error) {
	out, err := getData[WebhookEndpoint](ctx, s.client, http.MethodPost, "/webhooks", params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's webhook endpoints (secrets omitted).
func (s *WebhooksService) List(ctx context.Context) ([]WebhookEndpoint, error) {
	return getData[[]WebhookEndpoint](ctx, s.client, http.MethodGet, "/webhooks", nil)
}

// Delete removes a webhook endpoint.
func (s *WebhooksService) Delete(ctx context.Context, id string) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodDelete, "/webhooks/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Deliveries lists recent delivery attempts to a webhook endpoint, newest first.
func (s *WebhooksService) Deliveries(ctx context.Context, id string, params *DeliveryListParams) ([]EventDelivery, error) {
	path := fmt.Sprintf("/webhooks/%s/deliveries", id)
	if params != nil {
		q := newQuery().int("limit", params.Limit).str("status", params.Status)
		if params.Offset != 0 {
			q.int("offset", params.Offset)
		}
		path = q.apply(path)
	}
	return getData[[]EventDelivery](ctx, s.client, http.MethodGet, path, nil)
}
