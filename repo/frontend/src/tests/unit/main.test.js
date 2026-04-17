import { describe, it, expect, vi, beforeEach } from 'vitest';

/**
 * Smoke test for main.js — verifies the app bootstraps correctly with
 * router and store wired up, and mounts without errors.
 */

// Mock all API and router modules to prevent side effects.
vi.mock('@/api/client.js', () => ({
  default: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  setLogoutCallback: vi.fn(),
}));
vi.mock('@/api/auth.js', () => ({
  login: vi.fn(), logout: vi.fn(), refresh: vi.fn(),
}));
vi.mock('@/api/org.js', () => ({
  getTree: vi.fn().mockResolvedValue({ data: { data: [] } }),
  getNodes: vi.fn().mockResolvedValue({ data: { data: [] } }),
  getCurrentContext: vi.fn().mockResolvedValue({ data: { data: { current_node: null, scope_ids: [], breadcrumb: [] } } }),
  switchContext: vi.fn().mockResolvedValue({ data: {} }),
}));

// Stub all page components.
vi.mock('@/pages/LoginPage.vue', () => ({ default: { template: '<div>Login</div>' } }));
vi.mock('@/pages/OrgTreePage.vue', () => ({ default: { template: '<div>Org</div>' } }));
vi.mock('@/pages/MasterDataPage.vue', () => ({ default: { template: '<div>Master</div>' } }));
vi.mock('@/pages/PlaybackPage.vue', () => ({ default: { template: '<div>Playback</div>' } }));
vi.mock('@/pages/AnalyticsPage.vue', () => ({ default: { template: '<div>Analytics</div>' } }));
vi.mock('@/pages/IngestionPage.vue', () => ({ default: { template: '<div>Ingestion</div>' } }));
vi.mock('@/pages/ReportsPage.vue', () => ({ default: { template: '<div>Reports</div>' } }));
vi.mock('@/pages/SecurityAdminPage.vue', () => ({ default: { template: '<div>Security</div>' } }));

import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from '@/App.vue';
import router from '@/router/index.js';

describe('main.js bootstrap', () => {
  beforeEach(() => {
    // Create a fresh mount point for each test.
    const el = document.createElement('div');
    el.id = 'app';
    document.body.appendChild(el);
  });

  it('creates a Vue app instance without errors', () => {
    const app = createApp(App);
    expect(app).toBeDefined();
    expect(typeof app.use).toBe('function');
    expect(typeof app.mount).toBe('function');
  });

  it('wires up Pinia store and router', () => {
    const app = createApp(App);
    const pinia = createPinia();

    // These should not throw.
    app.use(pinia);
    app.use(router);

    expect(app).toBeDefined();
  });

  it('mounts to #app without throwing', async () => {
    const app = createApp(App);
    const pinia = createPinia();
    app.use(pinia);
    app.use(router);

    // Push to login so router guard doesn't redirect.
    router.push('/login');
    await router.isReady();

    const instance = app.mount('#app');
    expect(instance).toBeDefined();

    // Clean up.
    app.unmount();
  });
});
