import client from './client.js';

export function login(username, password) {
  return client.post('/auth/login', { username, password });
}

export function logout() {
  return client.post('/auth/logout');
}

export function refresh(refreshToken) {
  return client.post('/auth/refresh', { refreshToken });
}
