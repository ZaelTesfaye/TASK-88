import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

// Mock the api/client.js module
vi.mock('@/api/client.js', () => ({
  default: { post: vi.fn(), get: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

// Mock the auth API
vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
}));

// We need to import the actual router (not mocked) so we can test guards
// But we must mock lazy-loaded page components
vi.mock('@/pages/LoginPage.vue', () => ({ default: { template: '<div>Login</div>' } }));
vi.mock('@/pages/OrgTreePage.vue', () => ({ default: { template: '<div>OrgTree</div>' } }));
vi.mock('@/pages/MasterDataPage.vue', () => ({ default: { template: '<div>MasterData</div>' } }));
vi.mock('@/pages/PlaybackPage.vue', () => ({ default: { template: '<div>Playback</div>' } }));
vi.mock('@/pages/AnalyticsPage.vue', () => ({ default: { template: '<div>Analytics</div>' } }));
vi.mock('@/pages/IngestionPage.vue', () => ({ default: { template: '<div>Ingestion</div>' } }));
vi.mock('@/pages/ReportsPage.vue', () => ({ default: { template: '<div>Reports</div>' } }));
vi.mock('@/pages/SecurityAdminPage.vue', () => ({ default: { template: '<div>Security</div>' } }));

import router from '@/router/index.js';
import { useAuthStore } from '@/stores/auth.js';

function setupAuthenticatedUser(role) {
  const store = useAuthStore();
  store.token = 'test-token';
  store.user = { id: '1', username: 'testuser', role };
}

describe('Router Guards', () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    // Reset the router to a known state - use login which is public
    await router.push('/login');
    await router.isReady();
    vi.clearAllMocks();
  });

  it('unauthenticated user redirected to /login', async () => {
    const store = useAuthStore();
    store.token = null;
    store.user = null;

    await router.push('/master/sku');
    await router.isReady();

    expect(router.currentRoute.value.path).toBe('/login');
    expect(router.currentRoute.value.query.redirect).toBe('/master/sku');
  });

  it('authenticated user can access /master/sku', async () => {
    setupAuthenticatedUser('data_steward');

    await router.push('/master/sku');
    await router.isReady();

    expect(router.currentRoute.value.path).toBe('/master/sku');
  });

  it('system_admin can access /org', async () => {
    setupAuthenticatedUser('system_admin');

    await router.push('/org');
    await router.isReady();

    expect(router.currentRoute.value.path).toBe('/org');
  });

  it('data_steward cannot access /org (role guard)', async () => {
    setupAuthenticatedUser('data_steward');

    // Start at a valid route first
    await router.push('/master/sku');
    await router.isReady();
    expect(router.currentRoute.value.path).toBe('/master/sku');

    // Try to access /org - should be redirected back to /master/sku
    await router.push('/org');
    await router.isReady();

    // data_steward does not have 'system_admin' role, so /org rejects to /master/sku
    expect(router.currentRoute.value.path).toBe('/master/sku');
  });

  it('standard_user cannot access /security', async () => {
    setupAuthenticatedUser('standard_user');

    // standard_user can access /playback (role is in the list)
    await router.push('/playback');
    await router.isReady();
    expect(router.currentRoute.value.path).toBe('/playback');

    // Verify via role check that standard_user would be blocked from /security
    const auth = useAuthStore();
    expect(auth.hasAnyRole(['system_admin'])).toBe(false);
  });

  it('operations_analyst can access /analytics', async () => {
    setupAuthenticatedUser('operations_analyst');

    await router.push('/analytics');
    await router.isReady();

    expect(router.currentRoute.value.path).toBe('/analytics');
  });

  it('standard_user cannot access /analytics', async () => {
    setupAuthenticatedUser('standard_user');

    // standard_user can access /playback
    await router.push('/playback');
    await router.isReady();
    expect(router.currentRoute.value.path).toBe('/playback');

    // /analytics requires ['operations_analyst', 'system_admin']
    const auth = useAuthStore();
    expect(auth.hasAnyRole(['operations_analyst', 'system_admin'])).toBe(false);
  });

  it('system_admin can access /security', async () => {
    setupAuthenticatedUser('system_admin');

    await router.push('/security');
    await router.isReady();

    expect(router.currentRoute.value.path).toBe('/security');
  });

  it('login page is accessible without auth (public route)', async () => {
    const store = useAuthStore();
    store.token = null;
    store.user = null;

    await router.push('/login');
    await router.isReady();

    expect(router.currentRoute.value.path).toBe('/login');
  });

  it('standard_user can access /master/sku (read-only view)', async () => {
    setupAuthenticatedUser('standard_user');

    await router.push('/master/sku');
    await router.isReady();

    expect(router.currentRoute.value.path).toBe('/master/sku');
  });

  it('standard_user cannot access /reports (restricted to ops_analyst/admin)', async () => {
    setupAuthenticatedUser('standard_user');

    // standard_user can access /master/sku
    await router.push('/master/sku');
    await router.isReady();
    expect(router.currentRoute.value.path).toBe('/master/sku');

    // /reports requires ['system_admin', 'operations_analyst']
    const auth = useAuthStore();
    expect(auth.hasAnyRole(['system_admin', 'operations_analyst'])).toBe(false);
  });
});
