import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';

// Mock all API modules to prevent real HTTP calls.
vi.mock('@/api/client.js', () => ({
  default: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  setLogoutCallback: vi.fn(),
}));
vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
}));
vi.mock('@/api/org.js', () => ({
  getTree: vi.fn().mockResolvedValue({ data: { data: [] } }),
  getNodes: vi.fn().mockResolvedValue({ data: { data: [] } }),
  getCurrentContext: vi.fn().mockResolvedValue({
    data: { data: { current_node: null, scope_ids: [], breadcrumb: [] } },
  }),
  switchContext: vi.fn().mockResolvedValue({ data: {} }),
}));

// Stub all page components to lightweight divs.
vi.mock('@/pages/LoginPage.vue', () => ({
  default: { template: '<div class="page-login">LoginPage</div>' },
}));
vi.mock('@/pages/OrgTreePage.vue', () => ({
  default: { template: '<div class="page-org">OrgTreePage</div>' },
}));
vi.mock('@/pages/MasterDataPage.vue', () => ({
  default: { template: '<div class="page-master">MasterDataPage</div>' },
}));
vi.mock('@/pages/PlaybackPage.vue', () => ({
  default: { template: '<div class="page-playback">PlaybackPage</div>' },
}));
vi.mock('@/pages/AnalyticsPage.vue', () => ({
  default: { template: '<div class="page-analytics">AnalyticsPage</div>' },
}));
vi.mock('@/pages/IngestionPage.vue', () => ({
  default: { template: '<div class="page-ingestion">IngestionPage</div>' },
}));
vi.mock('@/pages/ReportsPage.vue', () => ({
  default: { template: '<div class="page-reports">ReportsPage</div>' },
}));
vi.mock('@/pages/SecurityAdminPage.vue', () => ({
  default: { template: '<div class="page-security">SecurityAdminPage</div>' },
}));

import App from '@/App.vue';
import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '@/stores/auth.js';

function makeRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/login', name: 'Login', component: () => import('@/pages/LoginPage.vue'), meta: { public: true } },
      { path: '/org', name: 'OrgTree', component: () => import('@/pages/OrgTreePage.vue'), meta: { roles: ['system_admin'] } },
      { path: '/master/:entity', name: 'MasterData', component: () => import('@/pages/MasterDataPage.vue'), meta: { roles: ['system_admin', 'data_steward', 'operations_analyst', 'standard_user'] } },
      { path: '/:pathMatch(.*)*', redirect: '/login' },
    ],
  });
}

describe('App.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('renders login layout when not authenticated', async () => {
    const router = makeRouter();
    router.push('/login');
    await router.isReady();

    const wrapper = mount(App, {
      global: {
        plugins: [router],
        stubs: {
          AppBreadcrumb: { template: '<nav />' },
          AppChip: { template: '<span />' },
          teleport: true,
        },
      },
    });
    await flushPromises();

    expect(wrapper.find('.auth-layout').exists()).toBe(true);
    expect(wrapper.find('.app-layout').exists()).toBe(false);
  });

  it('renders app layout with sidebar when authenticated', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);

    const router = makeRouter();
    router.push('/org');
    await router.isReady();

    const wrapper = mount(App, {
      global: {
        plugins: [router, pinia],
        stubs: {
          AppBreadcrumb: { template: '<nav />' },
          AppChip: { template: '<span />' },
          teleport: true,
        },
      },
    });

    // Simulate authentication.
    const authStore = useAuthStore();
    authStore.$patch({
      token: 'fake-token',
      user: { id: 1, username: 'admin', role: 'system_admin' },
    });

    await flushPromises();

    expect(wrapper.find('.app-layout').exists()).toBe(true);
    expect(wrapper.find('.sidebar').exists()).toBe(true);
    expect(wrapper.find('.topbar').exists()).toBe(true);
  });

  it('shows nav items based on user role', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);

    const router = makeRouter();
    router.push('/org');
    await router.isReady();

    const wrapper = mount(App, {
      global: {
        plugins: [router, pinia],
        stubs: {
          AppBreadcrumb: { template: '<nav />' },
          AppChip: { template: '<span />' },
          teleport: true,
        },
      },
    });

    const authStore = useAuthStore();
    authStore.$patch({
      token: 'fake-token',
      user: { id: 1, username: 'admin', role: 'system_admin' },
    });

    await flushPromises();

    const navItems = wrapper.findAll('.nav-item');
    // system_admin sees all nav items (Org, Master, Playback, Analytics, Ingestion, Reports, Security).
    expect(navItems.length).toBe(7);
  });
});
