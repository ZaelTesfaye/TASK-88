import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { login, logout, refresh } from '@/api/auth.js';

describe('api/auth.js', () => {
  let originalAdapter;
  let captured;

  beforeEach(() => {
    originalAdapter = client.defaults.adapter;
    captured = null;
  });
  afterEach(() => { client.defaults.adapter = originalAdapter; });

  function mockOK(data = {}) {
    client.defaults.adapter = (config) => {
      captured = config;
      return Promise.resolve({ data, status: 200, statusText: 'OK', headers: {}, config });
    };
  }

  it('login sends POST /auth/login with username and password', async () => {
    mockOK({ token: 'jwt', refreshToken: 'rt', user: { id: 1 } });
    const resp = await login('admin', 'pass');
    expect(captured.method).toBe('post');
    expect(captured.url).toBe('/auth/login');
    expect(JSON.parse(captured.data)).toEqual({ username: 'admin', password: 'pass' });
    expect(resp.data.token).toBe('jwt');
  });

  it('logout sends POST /auth/logout', async () => {
    mockOK({ message: 'logged out' });
    await logout();
    expect(captured.method).toBe('post');
    expect(captured.url).toBe('/auth/logout');
  });

  it('refresh sends POST /auth/refresh with refreshToken', async () => {
    mockOK({ token: 'new-jwt', refreshToken: 'new-rt' });
    const resp = await refresh('old-rt');
    expect(captured.method).toBe('post');
    expect(captured.url).toBe('/auth/refresh');
    expect(JSON.parse(captured.data)).toEqual({ refreshToken: 'old-rt' });
    expect(resp.data.token).toBe('new-jwt');
  });

  it('login propagates 401 error', async () => {
    client.defaults.adapter = (config) => {
      const err = new Error('Unauthorized');
      err.response = { status: 401, data: { code: 'AUTH_REQUIRED', message: 'invalid credentials' }, headers: {}, config };
      return Promise.reject(err);
    };
    await expect(login('bad', 'creds')).rejects.toMatchObject({ code: 'AUTH_REQUIRED' });
  });
});
