import client from './client.js';

export function listVersions(entity, params = {}) {
  return client.get(`/versions/${entity}`, { params });
}

export function getVersion(entity, versionId) {
  return client.get(`/versions/${entity}/${versionId}`);
}

export function getVersionItems(entity, versionId, params = {}) {
  return client.get(`/versions/${entity}/${versionId}/items`, { params });
}

export function diffVersions(entity, versionId, params = {}) {
  return client.get(`/versions/${entity}/${versionId}/diff`, { params });
}

export function createVersion(entity, payload) {
  return client.post(`/versions/${entity}`, payload);
}

export function submitReview(entity, versionId) {
  return client.post(`/versions/${entity}/${versionId}/review`);
}

export function addVersionItem(entity, versionId, payload) {
  return client.post(`/versions/${entity}/${versionId}/items`, payload);
}

export function removeVersionItem(entity, versionId, itemId) {
  return client.delete(`/versions/${entity}/${versionId}/items/${itemId}`);
}

export function activate(entity, versionId) {
  return client.post(`/versions/${entity}/${versionId}/activate`);
}
