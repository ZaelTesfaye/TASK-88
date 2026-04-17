import { describe, it, expect, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';

// Mock the api/client.js module so setLogoutCallback does not throw
vi.mock('@/api/client.js', () => ({
  default: { post: vi.fn(), get: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

// Mock the router so store creation doesn't blow up
vi.mock('@/router/index.js', () => ({
  default: { push: vi.fn() },
}));

// Mock the auth API
vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
}));

import { useAuthStore } from '@/stores/auth.js';
import * as authApi from '@/api/auth.js';
import router from '@/router/index.js';

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('login sets token and user in store', async () => {
    const mockResponse = {
      data: {
        token: 'test-jwt-token',
        refreshToken: 'test-refresh-token',
        user: { id: '1', username: 'admin', role: 'system_admin', permissions: ['read', 'write'] },
        defaultContext: { id: 'org-1', name: 'Root Org' },
      },
    };
    authApi.login.mockResolvedValue(mockResponse);

    const store = useAuthStore();
    await store.login('admin', 'Password123!');

    expect(store.token).toBe('test-jwt-token');
    expect(store.refreshToken).toBe('test-refresh-token');
    expect(store.user).toEqual(mockResponse.data.user);
    expect(store.currentContext).toEqual(mockResponse.data.defaultContext);
    expect(authApi.login).toHaveBeenCalledWith('admin', 'Password123!');
  });

  it('logout clears token and user', async () => {
    authApi.login.mockResolvedValue({
      data: {
        token: 'tok',
        refreshToken: 'ref',
        user: { id: '1', username: 'admin', role: 'system_admin' },
      },
    });
    authApi.logout.mockResolvedValue({});

    const store = useAuthStore();
    await store.login('admin', 'pass');
    expect(store.token).toBe('tok');

    await store.logout();

    expect(store.token).toBeNull();
    expect(store.user).toBeNull();
    expect(store.refreshToken).toBeNull();
    expect(store.currentContext).toBeNull();
    expect(router.push).toHaveBeenCalledWith('/login');
  });

  it('isAuthenticated getter returns true when token present', async () => {
    authApi.login.mockResolvedValue({
      data: {
        token: 'some-token',
        refreshToken: 'ref',
        user: { id: '1', username: 'user1', role: 'viewer' },
      },
    });

    const store = useAuthStore();
    expect(store.isAuthenticated).toBe(false);

    await store.login('user1', 'pass');
    expect(store.isAuthenticated).toBe(true);
  });

  it('userRole getter returns correct role', async () => {
    authApi.login.mockResolvedValue({
      data: {
        token: 'tok',
        refreshToken: 'ref',
        user: { id: '1', username: 'steward', role: 'data_steward' },
      },
    });

    const store = useAuthStore();
    expect(store.userRole).toBeNull();

    await store.login('steward', 'pass');
    expect(store.userRole).toBe('data_steward');
  });

  it('token persists to localStorage', async () => {
    authApi.login.mockResolvedValue({
      data: {
        token: 'persisted-token',
        refreshToken: 'persisted-refresh',
        user: { id: '1', username: 'admin', role: 'system_admin' },
      },
    });

    const store = useAuthStore();
    await store.login('admin', 'pass');

    expect(localStorage.getItem('auth_token')).toBe('persisted-token');
    expect(localStorage.getItem('auth_refresh_token')).toBe('persisted-refresh');
    expect(JSON.parse(localStorage.getItem('auth_user'))).toEqual({
      id: '1',
      username: 'admin',
      role: 'system_admin',
    });
  });

  it('logout clears localStorage', async () => {
    authApi.login.mockResolvedValue({
      data: {
        token: 'tok',
        refreshToken: 'ref',
        user: { id: '1', username: 'admin', role: 'system_admin' },
      },
    });
    authApi.logout.mockResolvedValue({});

    const store = useAuthStore();
    await store.login('admin', 'pass');
    expect(localStorage.getItem('auth_token')).toBe('tok');

    await store.logout();
    expect(localStorage.getItem('auth_token')).toBeNull();
    expect(localStorage.getItem('auth_refresh_token')).toBeNull();
    expect(localStorage.getItem('auth_user')).toBeNull();
  });

  it('hasRole returns true for matching role', async () => {
    authApi.login.mockResolvedValue({
      data: {
        token: 'tok',
        refreshToken: 'ref',
        user: { id: '1', username: 'admin', role: 'system_admin' },
      },
    });

    const store = useAuthStore();
    await store.login('admin', 'pass');

    expect(store.hasRole('system_admin')).toBe(true);
    expect(store.hasRole('viewer')).toBe(false);
  });

  it('hasAnyRole returns true when role in list', async () => {
    authApi.login.mockResolvedValue({
      data: {
        token: 'tok',
        refreshToken: 'ref',
        user: { id: '1', username: 'analyst', role: 'operations_analyst' },
      },
    });

    const store = useAuthStore();
    await store.login('analyst', 'pass');

    expect(store.hasAnyRole(['system_admin', 'operations_analyst'])).toBe(true);
    expect(store.hasAnyRole(['viewer', 'data_steward'])).toBe(false);
  });

  it('restoreSession loads state from localStorage', () => {
    localStorage.setItem('auth_token', 'stored-token');
    localStorage.setItem('auth_refresh_token', 'stored-refresh');
    localStorage.setItem('auth_user', JSON.stringify({ id: '5', username: 'stored', role: 'viewer' }));

    const store = useAuthStore();
    expect(store.token).toBe('stored-token');
    expect(store.user?.username).toBe('stored');
  });

  it('restoreSession clears state on corrupted user JSON', () => {
    localStorage.setItem('auth_token', 'stored-token');
    localStorage.setItem('auth_user', 'not-valid-json{{{');

    const store = useAuthStore();
    // Corrupted user JSON triggers clearAuth() inside restoreSession.
    expect(store.user).toBeNull();
  });

  it('login failure propagates error', async () => {
    authApi.login.mockRejectedValue({ code: 'AUTH_REQUIRED', message: 'bad creds' });
    const store = useAuthStore();
    await expect(store.login('x', 'y')).rejects.toMatchObject({ code: 'AUTH_REQUIRED' });
    expect(store.token).toBeNull();
  });
});
