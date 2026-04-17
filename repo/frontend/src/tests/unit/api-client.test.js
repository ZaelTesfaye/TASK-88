import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

/**
 * Tests for the real api/client.js module behavior.
 *
 * We override the axios adapter to intercept at the lowest level, allowing
 * the real interceptor chain (auth header injection, correlation ID, error
 * transform, 401 logout) to execute untouched.
 */

import client, { setLogoutCallback } from '@/api/client.js';

describe('api/client.js', () => {
  let originalAdapter;

  beforeEach(() => {
    localStorage.clear();
    // Save and replace the axios adapter so no real network calls happen.
    originalAdapter = client.defaults.adapter;
  });

  afterEach(() => {
    client.defaults.adapter = originalAdapter;
  });

  function mockAdapter(fn) {
    client.defaults.adapter = (config) => fn(config);
  }

  it('attaches Authorization header when token exists in localStorage', async () => {
    localStorage.setItem('auth_token', 'test-jwt-token');

    let capturedConfig;
    mockAdapter((config) => {
      capturedConfig = config;
      return Promise.resolve({ data: {}, status: 200, statusText: 'OK', headers: {}, config });
    });

    await client.get('/test');
    expect(capturedConfig.headers.Authorization).toBe('Bearer test-jwt-token');
  });

  it('omits Authorization header when no token in localStorage', async () => {
    let capturedConfig;
    mockAdapter((config) => {
      capturedConfig = config;
      return Promise.resolve({ data: {}, status: 200, statusText: 'OK', headers: {}, config });
    });

    await client.get('/test');
    expect(capturedConfig.headers.Authorization).toBeUndefined();
  });

  it('attaches X-Correlation-ID header to every request', async () => {
    let capturedConfig;
    mockAdapter((config) => {
      capturedConfig = config;
      return Promise.resolve({ data: {}, status: 200, statusText: 'OK', headers: {}, config });
    });

    await client.get('/test');
    expect(capturedConfig.headers['X-Correlation-ID']).toBeDefined();
    expect(capturedConfig.headers['X-Correlation-ID'].length).toBeGreaterThan(0);
  });

  it('wraps network errors into a standardized error shape', async () => {
    mockAdapter(() => {
      return Promise.reject(new Error('Network Error'));
    });

    try {
      await client.get('/test');
      expect.fail('should have thrown');
    } catch (err) {
      expect(err.code).toBe('NETWORK_ERROR');
      expect(err.message).toContain('Unable to reach the server');
    }
  });

  it('transforms HTTP error responses into standardized error shape', async () => {
    mockAdapter((config) => {
      const error = new Error('Bad Request');
      error.response = {
        status: 400,
        data: { code: 'BAD_REQUEST', message: 'invalid input' },
        headers: { 'x-correlation-id': 'test-corr-id' },
        config,
        statusText: 'Bad Request',
      };
      return Promise.reject(error);
    });

    try {
      await client.get('/test');
      expect.fail('should have thrown');
    } catch (err) {
      expect(err.code).toBe('BAD_REQUEST');
      expect(err.message).toBe('invalid input');
      expect(err.status).toBe(400);
      expect(err.correlationId).toBe('test-corr-id');
    }
  });

  it('clears localStorage and calls logoutCallback on 401', async () => {
    localStorage.setItem('auth_token', 'expired-token');
    localStorage.setItem('auth_refresh_token', 'old-refresh');
    localStorage.setItem('auth_user', '{"id":1}');

    const logoutFn = vi.fn();
    setLogoutCallback(logoutFn);

    mockAdapter((config) => {
      const error = new Error('Unauthorized');
      error.response = {
        status: 401,
        data: { code: 'AUTH_REQUIRED', message: 'token expired' },
        headers: {},
        config,
        statusText: 'Unauthorized',
      };
      return Promise.reject(error);
    });

    try {
      await client.get('/test');
    } catch {
      // Expected rejection.
    }

    expect(localStorage.getItem('auth_token')).toBeNull();
    expect(localStorage.getItem('auth_refresh_token')).toBeNull();
    expect(localStorage.getItem('auth_user')).toBeNull();
    expect(logoutFn).toHaveBeenCalledTimes(1);
  });

  it('sets correct baseURL and timeout defaults', () => {
    expect(client.defaults.baseURL).toBeDefined();
    expect(client.defaults.timeout).toBe(30000);
    expect(client.defaults.headers['Content-Type']).toBe('application/json');
  });
});
