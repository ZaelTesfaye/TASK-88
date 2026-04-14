import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn().mockResolvedValue({}),
  refresh: vi.fn(),
}));

vi.mock('@/api/client.js', () => ({
  default: { post: vi.fn(), get: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

vi.mock('@/api/org.js', () => ({
  switchContext: vi.fn(),
  getOrgTree: vi.fn(),
}));

vi.mock('@/router/index.js', () => ({
  default: { push: vi.fn() },
}));

const mockPush = vi.fn();
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useRoute: () => ({ query: {} }),
  RouterLink: { template: '<a class="nav-item"><slot /></a>', props: ['to'] },
  RouterView: { template: '<div />' },
}));

import App from '@/App.vue';
import { useAuthStore } from '@/stores/auth.js';
import { useContextStore } from '@/stores/context.js';

function createWrapper(role = 'system_admin') {
  const pinia = createPinia();
  setActivePinia(pinia);

  const authStore = useAuthStore();
  authStore.token = 'test-token';
  authStore.user = { id: 'u1', username: 'testadmin', role };

  const contextStore = useContextStore();
  contextStore.breadcrumb = [
    { id: 'n1', name: 'Global Corp', level: 'L1' },
    { id: 'n2', name: 'North America', level: 'L2' },
  ];

  return shallowMount(App, {
    global: {
      plugins: [pinia],
      stubs: {
        AppBreadcrumb: {
          template: '<div class="app-breadcrumb">{{ items.map(i => i.label).join(" > ") }}</div>',
          props: ['items'],
        },
        AppChip: { template: '<span class="app-chip">{{ label }}</span>', props: ['status', 'label', 'size'] },
        RouterLink: { template: '<a :href="to" class="nav-item"><slot /></a>', props: ['to', 'activeClass'] },
        RouterView: { template: '<div />' },
      },
    },
  });
}

describe('App Layout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Sidebar Navigation by Role', () => {
    it('shows all nav items for system_admin', () => {
      const wrapper = createWrapper('system_admin');
      const items = wrapper.vm.visibleNavItems;
      const labels = items.map(i => i.label);

      expect(labels).toContain('Org Tree');
      expect(labels).toContain('Master Data');
      expect(labels).toContain('Ingestion');
      expect(labels).toContain('Reports');
      expect(labels).toContain('Security Admin');
      expect(labels).toContain('Analytics');
      expect(labels).toContain('Playback');
    });

    it('shows limited nav items for viewer', () => {
      const wrapper = createWrapper('viewer');
      const items = wrapper.vm.visibleNavItems;
      const labels = items.map(i => i.label);

      expect(labels).toContain('Playback');
      expect(labels).not.toContain('Org Tree');
      expect(labels).not.toContain('Ingestion');
      expect(labels).not.toContain('Security Admin');
      expect(labels).not.toContain('Reports');
    });

    it('shows correct items for operations_analyst', () => {
      const wrapper = createWrapper('operations_analyst');
      const items = wrapper.vm.visibleNavItems;
      const labels = items.map(i => i.label);

      expect(labels).toContain('Master Data');
      expect(labels).toContain('Analytics');
      expect(labels).toContain('Ingestion');
      expect(labels).toContain('Reports');
      expect(labels).not.toContain('Org Tree');
      expect(labels).not.toContain('Security Admin');
    });

    it('shows correct items for data_steward', () => {
      const wrapper = createWrapper('data_steward');
      const items = wrapper.vm.visibleNavItems;
      const labels = items.map(i => i.label);

      expect(labels).toContain('Master Data');
      expect(labels).not.toContain('Org Tree');
      expect(labels).not.toContain('Security Admin');
      expect(labels).not.toContain('Analytics');
    });
  });

  describe('Sidebar Collapse', () => {
    it('starts expanded by default', () => {
      const wrapper = createWrapper();
      expect(wrapper.vm.sidebarCollapsed).toBe(false);
    });

    it('adds collapsed class when toggled', async () => {
      const wrapper = createWrapper();

      wrapper.vm.sidebarCollapsed = true;
      await nextTick();

      expect(wrapper.find('.sidebar').classes()).toContain('collapsed');
    });

    it('hides logo text when collapsed', async () => {
      const wrapper = createWrapper();

      wrapper.vm.sidebarCollapsed = true;
      await nextTick();

      expect(wrapper.find('.logo-text').exists()).toBe(false);
    });

    it('shows logo text when expanded', () => {
      const wrapper = createWrapper();
      expect(wrapper.find('.logo-text').exists()).toBe(true);
      expect(wrapper.find('.logo-text').text()).toBe('Multi-Org Hub');
    });
  });

  describe('Context Breadcrumb', () => {
    it('renders breadcrumb from context store', () => {
      createWrapper();
      const contextStore = useContextStore();

      const crumbs = contextStore.contextBreadcrumb;
      expect(crumbs).toHaveLength(2);
      expect(crumbs[0].label).toBe('Global Corp');
      expect(crumbs[1].label).toBe('North America');
    });

    it('updates breadcrumb when context changes', async () => {
      createWrapper();
      const contextStore = useContextStore();

      contextStore.breadcrumb = [
        { id: 'n3', name: 'Europe', level: 'L2' },
      ];
      await nextTick();

      const crumbs = contextStore.contextBreadcrumb;
      expect(crumbs).toHaveLength(1);
      expect(crumbs[0].label).toBe('Europe');
    });
  });

  describe('Logout', () => {
    it('calls authStore.logout and closes user menu', async () => {
      const wrapper = createWrapper();
      const authStore = useAuthStore();
      authStore.logout = vi.fn().mockResolvedValue({});

      await wrapper.vm.handleLogout();
      await flushPromises();

      expect(authStore.logout).toHaveBeenCalledTimes(1);
      expect(wrapper.vm.userMenuOpen).toBe(false);
    });

    it('shows login layout when not authenticated', async () => {
      const pinia = createPinia();
      setActivePinia(pinia);
      const authStore = useAuthStore();
      authStore.token = null;
      authStore.user = null;

      const wrapper = shallowMount(App, {
        global: {
          plugins: [pinia],
          stubs: {
            AppBreadcrumb: { template: '<div />', props: ['items'] },
            AppChip: { template: '<span />', props: ['status', 'label', 'size'] },
            RouterLink: { template: '<a><slot /></a>', props: ['to'] },
            RouterView: { template: '<div />' },
          },
        },
      });

      expect(wrapper.find('.auth-layout').exists()).toBe(true);
      expect(wrapper.find('.app-layout').exists()).toBe(false);
    });
  });
});