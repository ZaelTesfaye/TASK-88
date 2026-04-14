import client from './client.js';

export function getMasterRecords(entity, params = {}) {
  return client.get(`/master/${entity}`, { params });
}

export function createRecord(entity, payload) {
  return client.post(`/master/${entity}`, payload);
}

export function updateRecord(entity, id, payload) {
  return client.put(`/master/${entity}/${id}`, payload);
}

export function deactivateRecord(entity, id, reason) {
  return client.post(`/master/${entity}/${id}/deactivate`, { reason });
}

export function importRecords(entity, file, onProgress) {
  const formData = new FormData();
  formData.append('file', file);
  return client.post(`/master/${entity}/import`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: onProgress
      ? (e) => onProgress(Math.round((e.loaded * 100) / e.total))
      : undefined,
  });
}

export function getDuplicates(entity, params = {}) {
  return client.get(`/master/${entity}/duplicates`, { params });
}
