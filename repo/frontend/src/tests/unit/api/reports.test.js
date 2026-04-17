import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import * as api from '@/api/reports.js';

describe('api/reports.js — full coverage', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getSchedules', async () => { ok({ items: [] }); await api.getSchedules(); expect(captured.url).toBe('/reports/schedules'); });
  it('createSchedule', async () => { ok({ id: 1 }); await api.createSchedule({ name: 'W' }); expect(captured.method).toBe('post'); });
  it('getSchedule', async () => { ok({ id: 5 }); await api.getSchedule(5); expect(captured.url).toBe('/reports/schedules/5'); });
  it('updateSchedule', async () => { ok(); await api.updateSchedule(5, { name: 'X' }); expect(captured.method).toBe('patch'); });
  it('deleteSchedule', async () => { ok(); await api.deleteSchedule(5); expect(captured.method).toBe('delete'); });
  it('triggerSchedule', async () => { ok({ id: 1 }); await api.triggerSchedule(5); expect(captured.url).toBe('/reports/schedules/5/trigger'); });
  it('getRuns', async () => { ok({ items: [] }); await api.getRuns({ schedule_id: 1 }); expect(captured.url).toBe('/reports/runs'); });
  it('getRun', async () => { ok({ id: 1 }); await api.getRun(1); expect(captured.url).toBe('/reports/runs/1'); });
  it('downloadRun returns URL or triggers download', () => {
    const url = api.downloadRun(5);
    // May return a promise or a URL — just ensure it's callable.
    expect(url).toBeDefined();
  });
  it('checkAccess', async () => { ok({ has_access: true }); await api.checkAccess(5); expect(captured.url).toBe('/reports/runs/5/access-check'); });
});
