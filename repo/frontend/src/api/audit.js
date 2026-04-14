import client from './client.js';

export function getLogs(params = {}) {
  return client.get('/audit/logs', { params });
}

export function getLog(logId) {
  return client.get(`/audit/logs/${logId}`);
}

export function searchLogs(params = {}) {
  return client.get('/audit/logs/search', { params });
}

export function getDeleteRequests(params = {}) {
  return client.get('/audit/delete-requests', { params });
}

export function createDeleteRequest(payload) {
  return client.post('/audit/delete-requests', payload);
}

export function getDeleteRequest(requestId) {
  return client.get(`/audit/delete-requests/${requestId}`);
}

export function approveDeleteRequest(requestId) {
  return client.post(`/audit/delete-requests/${requestId}/approve`);
}

export function executeDeleteRequest(requestId) {
  return client.post(`/audit/delete-requests/${requestId}/execute`);
}
