# Spec: Incident Alerting & Runbook

## Objective
Establish a baseline for operational maturity by implementing health-alert hooks (sending a webhook or email when the `/health` endpoint degrades) and documenting a SEV1 money-movement incident runbook before the first cloud customer is onboarded.

## Tech Stack
- Go 1.25+
- External Uptime Monitor (e.g., UptimeRobot, BetterStack) or internal cron
- Markdown (for the Runbook)

## Commands
Check health: `curl http://localhost:8080/health`
Lint Runbook: `markdownlint docs/incident-runbook.md`

## Project Structure
```
internal/
  adapter/
    handler/
      health.go           → Update to check DB and TigerBeetle connectivity
docs/
  incident-runbook.md     → Detailed steps for SEV1 mitigation
```

## Code Style
```go
// HealthCheck handler
func (h *HealthHandler) Check(c *gin.Context) {
	// 1. Check PostgreSQL
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "component": "postgres"})
		return
	}
	
	// 2. Check TigerBeetle
	// Assuming a ping/dummy query exists
	
	c.JSON(http.StatusOK, gin.H{"status": "up"})
}
```

## Testing Strategy
- **Chaos Testing**: Temporarily kill the PostgreSQL container during a local run and verify that `/health` returns `503 Service Unavailable`.
- **Runbook Drills**: Execute a mock "tabletop" exercise where a developer reads the runbook and successfully identifies the commands needed to roll back a deployment or pause billing.

## Boundaries
- **Always**: Keep the `/health` check fast. It should not execute complex queries that could further degrade a struggling database.
- **Ask first**: Before integrating heavy alerting solutions like PagerDuty into the open-source codebase. Alerting should ideally be handled by the deployment infrastructure (Kubernetes probes / Uptime monitors) rather than the application code itself calling Slack.
- **Never**: Expose sensitive system configuration or connection strings in the `/health` endpoint response.

## Success Criteria
- [ ] The `/health` endpoint correctly validates connectivity to both PostgreSQL and TigerBeetle.
- [ ] An external uptime monitor is configured to ping `/health` and send an alert if it fails.
- [ ] `docs/incident-runbook.md` exists and details steps to mitigate 1) Database corruption, 2) Gateway double-charging, and 3) Webhook loops.

## Open Questions
- Should the application itself contain logic to send a Slack webhook on startup/failure, or should we rely entirely on the external orchestrator (Kubernetes/Docker) and uptime monitors to handle notifications?
