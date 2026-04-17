import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import * as api from '@/api/security.js';

describe('api/security.js — full coverage', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getSensitiveFields', async () => { ok({ items: [] }); await api.getSensitiveFields({ page: 1 }); expect(captured.url).toBe('/security/sensitive-fields'); });
  it('createSensitiveField', async () => { ok({ id: 1 }); await api.createSensitiveField({ field_key: 'ssn' }); expect(captured.method).toBe('post'); });
  it('updateSensitiveField', async () => { ok(); await api.updateSensitiveField(3, { mask_pattern: 'X' }); expect(captured.url).toBe('/security/sensitive-fields/3'); expect(captured.method).toBe('put'); });
  it('deleteSensitiveField', async () => { ok(); await api.deleteSensitiveField(3); expect(captured.method).toBe('delete'); });
  it('getKeys', async () => { ok({ items: [] }); await api.getKeys(); expect(captured.url).toBe('/security/keys'); });
  it('getKey', async () => { ok({ id: 1 }); await api.getKey(1); expect(captured.url).toBe('/security/keys/1'); });
  it('rotateKey', async () => { ok(); await api.rotateKey({}); expect(captured.url).toBe('/security/keys/rotate'); expect(captured.method).toBe('post'); });
  it('createPasswordResetRequest', async () => { ok({ id: 1 }); await api.createPasswordResetRequest({ user_id: 2 }); expect(captured.url).toBe('/security/password-reset'); expect(captured.method).toBe('post'); });
  it('approvePasswordResetRequest', async () => { ok({ token: 'x' }); await api.approvePasswordResetRequest(5); expect(captured.url).toBe('/security/password-reset/5/approve'); });
  it('getPasswordResetRequests', async () => { ok({ items: [] }); await api.getPasswordResetRequests(); expect(captured.url).toBe('/security/password-reset'); });
  it('getRetentionPolicies', async () => { ok({ items: [] }); await api.getRetentionPolicies(); expect(captured.url).toBe('/security/retention-policies'); });
  it('createRetentionPolicy', async () => { ok({ id: 1 }); await api.createRetentionPolicy({ artifact_type: 'audit_logs' }); expect(captured.method).toBe('post'); });
  it('updateRetentionPolicy', async () => { ok(); await api.updateRetentionPolicy(3, { retention_days: 365 }); expect(captured.url).toBe('/security/retention-policies/3'); expect(captured.method).toBe('put'); });
  it('getLegalHolds', async () => { ok({ items: [] }); await api.getLegalHolds(); expect(captured.url).toBe('/security/legal-holds'); });
  it('createLegalHold', async () => { ok({ id: 1 }); await api.createLegalHold({ reason: 'test' }); expect(captured.method).toBe('post'); });
  it('releaseLegalHold', async () => { ok(); await api.releaseLegalHold(7); expect(captured.url).toBe('/security/legal-holds/7/release'); });
  it('dryRunPurge', async () => { ok({}); await api.dryRunPurge({ artifact_type: 'audit' }); expect(captured.url).toBe('/security/purge-runs/dry-run'); expect(captured.method).toBe('post'); });
  it('executePurge', async () => { ok({}); await api.executePurge({ artifact_type: 'audit' }); expect(captured.url).toBe('/security/purge-runs/execute'); expect(captured.method).toBe('post'); });
  it('getPurgeRuns', async () => { ok({ items: [] }); await api.getPurgeRuns(); expect(captured.url).toBe('/security/purge-runs'); });
  it('updateRetentionPolicies', async () => { ok(); await api.updateRetentionPolicies(3, { retention_days: 90 }); expect(captured.url).toBe('/security/retention-policies/3'); });
  it('updateSensitiveFields with id', async () => { ok(); await api.updateSensitiveFields({ id: 3, mask_pattern: 'X' }); expect(captured.url).toBe('/security/sensitive-fields/3'); });
});
