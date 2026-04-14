import client from './client.js';

export function getSchedules(params = {}) {
  return client.get('/reports/schedules', { params });
}

export function createSchedule(payload) {
  return client.post('/reports/schedules', payload);
}

export function getSchedule(scheduleId) {
  return client.get(`/reports/schedules/${scheduleId}`);
}

export function updateSchedule(scheduleId, payload) {
  return client.patch(`/reports/schedules/${scheduleId}`, payload);
}

export function deleteSchedule(scheduleId) {
  return client.delete(`/reports/schedules/${scheduleId}`);
}

export function triggerSchedule(scheduleId) {
  return client.post(`/reports/schedules/${scheduleId}/trigger`);
}

export function getRuns(params = {}) {
  return client.get('/reports/runs', { params });
}

export function getRun(runId) {
  return client.get(`/reports/runs/${runId}`);
}

export function downloadRun(runId) {
  return client.get(`/reports/runs/${runId}/download`, { responseType: 'blob' });
}

export function checkAccess(runId) {
  return client.get(`/reports/runs/${runId}/access-check`);
}
