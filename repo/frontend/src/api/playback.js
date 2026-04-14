import client from './client.js';

export function getMedia(params = {}) {
  return client.get('/media', { params });
}

export function createMedia(payload) {
  return client.post('/media', payload);
}

export function getMediaById(mediaId) {
  return client.get(`/media/${mediaId}`);
}

export function updateMedia(mediaId, payload) {
  return client.put(`/media/${mediaId}`, payload);
}

export function deleteMedia(mediaId) {
  return client.delete(`/media/${mediaId}`);
}

export function streamAudio(mediaId) {
  return client.get(`/media/${mediaId}/stream`, { responseType: 'blob' });
}

export function getCoverArt(mediaId) {
  return client.get(`/media/${mediaId}/cover`, { responseType: 'blob' });
}

export function parseLyrics(mediaId) {
  return client.post(`/media/${mediaId}/lyrics/parse`);
}

export function searchLyrics(mediaId, query) {
  return client.get(`/media/${mediaId}/lyrics/search`, { params: { q: query } });
}

export function getSupportedFormats() {
  return client.get('/media/formats/supported');
}
