import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getSensitiveFields, createSensitiveField, getKeys, rotateKey } from '@/api/security.js';

describe('api/security.js', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getSensitiveFields sends GET /security/sensitive-fields', async () => { ok({ items: [] }); await getSensitiveFields(); expect(captured.url).toBe('/security/sensitive-fields'); });
  it('createSensitiveField sends POST /security/sensitive-fields', async () => { ok({ id: 1 }); await createSensitiveField({ field_key: 'ssn' }); expect(captured.method).toBe('post'); expect(captured.url).toBe('/security/sensitive-fields'); });
  it('getKeys sends GET /security/keys', async () => { ok({ items: [] }); await getKeys(); expect(captured.url).toBe('/security/keys'); });
  it('rotateKey sends POST /security/keys/rotate', async () => { ok({ message: 'rotated' }); await rotateKey(); expect(captured.url).toBe('/security/keys/rotate'); expect(captured.method).toBe('post'); });
});
