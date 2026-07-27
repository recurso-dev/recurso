// Recurso telemetry receiver — the collector behind telemetry.recurso.dev.
//
// The product's opt-in anonymous telemetry (internal/adapter/telemetry) POSTs
// one small JSON event at a time: {event, instance_id, version, timestamp,
// ...coarse props}. This worker validates and stores them in D1, and serves a
// token-gated aggregate stats endpoint. It stores nothing beyond what the
// client is documented to send (docs/telemetry.mdx): no IPs, no user agents,
// no PII — the row is the payload plus a server receive time.

/** Milestones + lifecycle events the client sends today. Unknown-but-sane
 * event names are still accepted (forward compatibility with older/newer
 * instance versions); this list only drives the stats rollup. */
const KNOWN_EVENTS = [
  'boot',
  'heartbeat',
  'first_plan',
  'first_customer',
  'first_invoice',
  'first_payment',
];

const MAX_BODY_BYTES = 4096;
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const EVENT_RE = /^[a-z][a-z0-9_]{0,63}$/;

/**
 * validateEvent checks a parsed payload and returns {ok, error, row}.
 * Exported for tests. The row keeps indexed columns small and typed; every
 * other (coarse, documented) property lands in the props JSON blob.
 */
export function validateEvent(payload) {
  if (payload === null || typeof payload !== 'object' || Array.isArray(payload)) {
    return { ok: false, error: 'body must be a JSON object' };
  }
  const { event, instance_id: instanceID, version, timestamp, ...rest } = payload;
  if (typeof event !== 'string' || !EVENT_RE.test(event)) {
    return { ok: false, error: 'event must be a short snake_case name' };
  }
  if (typeof instanceID !== 'string' || !UUID_RE.test(instanceID)) {
    return { ok: false, error: 'instance_id must be a UUID' };
  }
  if (version !== undefined && (typeof version !== 'string' || version.length > 64)) {
    return { ok: false, error: 'version must be a string of at most 64 chars' };
  }
  if (timestamp !== undefined && (typeof timestamp !== 'string' || Number.isNaN(Date.parse(timestamp)))) {
    return { ok: false, error: 'timestamp must be RFC3339' };
  }
  // Props: flat, short, scalar-only — matches the client's coarse contract.
  const props = {};
  for (const [k, v] of Object.entries(rest)) {
    if (!EVENT_RE.test(k)) return { ok: false, error: `unexpected property key ${JSON.stringify(k)}` };
    const t = typeof v;
    if (t !== 'string' && t !== 'number' && t !== 'boolean') {
      return { ok: false, error: `property ${k} must be a scalar` };
    }
    if (t === 'string' && v.length > 128) return { ok: false, error: `property ${k} too long` };
    props[k] = v;
  }
  return {
    ok: true,
    row: {
      event,
      instance_id: instanceID.toLowerCase(),
      version: version ?? '',
      client_ts: timestamp ?? '',
      props: JSON.stringify(props),
    },
  };
}

async function handleIngest(request, env) {
  if (request.headers.get('content-type')?.split(';')[0]?.trim() !== 'application/json') {
    return json({ error: 'content-type must be application/json' }, 415);
  }
  const raw = await request.text();
  if (raw.length > MAX_BODY_BYTES) return json({ error: 'body too large' }, 413);
  let payload;
  try {
    payload = JSON.parse(raw);
  } catch {
    return json({ error: 'invalid JSON' }, 400);
  }
  const v = validateEvent(payload);
  if (!v.ok) return json({ error: v.error }, 400);

  await env.DB.prepare(
    `INSERT INTO events (received_at, event, instance_id, version, client_ts, props)
     VALUES (?1, ?2, ?3, ?4, ?5, ?6)`
  )
    .bind(new Date().toISOString(), v.row.event, v.row.instance_id, v.row.version, v.row.client_ts, v.row.props)
    .run();
  // 202: accepted, fire-and-forget on the client side.
  return json({ status: 'accepted' }, 202);
}

async function handleStats(request, env) {
  const auth = request.headers.get('authorization') ?? '';
  if (!env.STATS_TOKEN || auth !== `Bearer ${env.STATS_TOKEN}`) {
    return json({ error: 'unauthorized' }, 401);
  }
  const one = async (sql, ...binds) => (await env.DB.prepare(sql).bind(...binds).first()) ?? {};
  const all = async (sql, ...binds) => (await env.DB.prepare(sql).bind(...binds).all()).results ?? [];

  const since30 = new Date(Date.now() - 30 * 864e5).toISOString();
  const since7 = new Date(Date.now() - 7 * 864e5).toISOString();

  const totals = await one(
    `SELECT COUNT(*) AS events, COUNT(DISTINCT instance_id) AS instances FROM events`
  );
  const active30 = await one(
    `SELECT COUNT(DISTINCT instance_id) AS n FROM events WHERE received_at >= ?1`, since30
  );
  const active7 = await one(
    `SELECT COUNT(DISTINCT instance_id) AS n FROM events WHERE received_at >= ?1 AND event = 'heartbeat'`, since7
  );
  const milestones = await all(
    `SELECT event, COUNT(DISTINCT instance_id) AS instances FROM events
     WHERE event IN (${KNOWN_EVENTS.map(() => '?').join(',')})
     GROUP BY event ORDER BY instances DESC`, ...KNOWN_EVENTS
  );
  const versions = await all(
    `SELECT version, COUNT(DISTINCT instance_id) AS instances FROM events
     WHERE received_at >= ?1 GROUP BY version ORDER BY instances DESC LIMIT 10`, since30
  );

  return json({
    totals,
    instances_active_30d: active30.n ?? 0,
    instances_heartbeating_7d: active7.n ?? 0,
    milestones,
    versions_30d: versions,
  });
}

function json(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    if (url.pathname === '/v1/events' && request.method === 'POST') {
      try {
        return await handleIngest(request, env);
      } catch (err) {
        // Never make an opted-in instance's life worse: log and 500 quietly.
        console.log('ingest error', String(err));
        return json({ error: 'internal' }, 500);
      }
    }
    if (url.pathname === '/v1/stats' && request.method === 'GET') {
      return handleStats(request, env);
    }
    if (url.pathname === '/' || url.pathname === '/healthz') {
      return json({ service: 'recurso-telemetry', ok: true });
    }
    return json({ error: 'not found' }, 404);
  },
};
