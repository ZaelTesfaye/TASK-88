import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getSchedules, createSchedule, getSchedule, updateSchedule, deleteSchedule, triggerSchedule } from '@/api/reports.js';

describe('api/reports.js', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getSchedules sends GET /reports/schedules', async () => { ok({ items: [] }); await getSchedules({ page: 1 }); expect(captured.url).toBe('/reports/schedules'); });
  it('createSchedule sends POST /reports/schedules', async () => { ok({ id: 1 }); await createSchedule({ name: 'W', cron_expr: '0 8 * * 1' }); expect(captured.method).toBe('post'); expect(captured.url).toBe('/reports/schedules'); });
  it('getSchedule sends GET /reports/schedules/:id', async () => { ok({ id: 5 }); await getSchedule(5); expect(captured.url).toBe('/reports/schedules/5'); });
  it('updateSchedule sends PATCH /reports/schedules/:id', async () => { ok(); await updateSchedule(5, { name: 'X' }); expect(captured.method).toBe('patch'); expect(captured.url).toBe('/reports/schedules/5'); });
  it('deleteSchedule sends DELETE /reports/schedules/:id', async () => { ok(); await deleteSchedule(5); expect(captured.method).toBe('delete'); expect(captured.url).toBe('/reports/schedules/5'); });
  it('triggerSchedule sends POST /reports/schedules/:id/trigger', async () => { ok({ id: 1 }); await triggerSchedule(5); expect(captured.url).toBe('/reports/schedules/5/trigger'); });
});
