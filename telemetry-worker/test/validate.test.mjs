import { test } from 'node:test';
import assert from 'node:assert/strict';
import { validateEvent } from '../src/worker.js';

const good = {
  event: 'first_invoice',
  instance_id: '2b1444a4-1698-4b19-b6b1-84a29d2a7b56',
  version: 'v0.9.0',
  timestamp: '2026-07-27T10:00:00Z',
  deployment: 'docker',
};

test('accepts the documented client payload', () => {
  const v = validateEvent(good);
  assert.equal(v.ok, true);
  assert.equal(v.row.event, 'first_invoice');
  assert.equal(v.row.instance_id, '2b1444a4-1698-4b19-b6b1-84a29d2a7b56');
  assert.deepEqual(JSON.parse(v.row.props), { deployment: 'docker' });
});

test('accepts heartbeats with bucketed counts', () => {
  const v = validateEvent({ ...good, event: 'heartbeat', tenants: '1-9', subscriptions: '10-99' });
  assert.equal(v.ok, true);
  assert.deepEqual(JSON.parse(v.row.props), { deployment: 'docker', tenants: '1-9', subscriptions: '10-99' });
});

test('rejects a missing/invalid instance_id (anonymity is the contract)', () => {
  assert.equal(validateEvent({ ...good, instance_id: undefined }).ok, false);
  assert.equal(validateEvent({ ...good, instance_id: 'my-hostname' }).ok, false);
});

test('rejects bad event names', () => {
  assert.equal(validateEvent({ ...good, event: 'DROP TABLE' }).ok, false);
  assert.equal(validateEvent({ ...good, event: 'x'.repeat(80) }).ok, false);
});

test('rejects nested/oversized props (coarse scalars only)', () => {
  assert.equal(validateEvent({ ...good, extra: { nested: true } }).ok, false);
  assert.equal(validateEvent({ ...good, extra: 'y'.repeat(200) }).ok, false);
  assert.equal(validateEvent({ ...good, 'bad key!': 'v' }).ok, false);
});

test('rejects non-object bodies', () => {
  assert.equal(validateEvent([good]).ok, false);
  assert.equal(validateEvent(null).ok, false);
  assert.equal(validateEvent('str').ok, false);
});
