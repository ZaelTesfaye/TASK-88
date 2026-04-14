import client from './client.js';

export function getKPIs(params = {}) {
  return client.get('/analytics/kpis', { params });
}

export function getTrends(params = {}) {
  return client.get('/analytics/trends', { params });
}
