import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

// Mock echarts to prevent canvas rendering issues in jsdom.
vi.mock('echarts', () => ({
  init: vi.fn(() => ({
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn(),
    on: vi.fn(),
    off: vi.fn(),
    dispatchAction: vi.fn(),
    getDataURL: vi.fn(() => ''),
  })),
  default: { init: vi.fn() },
}));

vi.mock('@/api/client.js', () => ({
  default: { get: vi.fn(), post: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

vi.mock('@/api/analytics.js', () => ({
  getKPIs: vi.fn().mockResolvedValue({ data: { items: [] } }),
  getKPIDefinitions: vi.fn().mockResolvedValue({ data: { items: [], total: 0 } }),
  createKPIDefinition: vi.fn().mockResolvedValue({ data: { id: 1, code: 'test' } }),
  getKPIDefinition: vi.fn(),
  updateKPIDefinition: vi.fn(),
  deleteKPIDefinition: vi.fn(),
  getTrends: vi.fn().mockResolvedValue({ data: { series: [] } }),
}));

vi.mock('@/api/org.js', () => ({
  getTree: vi.fn().mockResolvedValue({ data: { data: [] } }),
  getNodes: vi.fn().mockResolvedValue({ data: { data: [] } }),
  getCurrentContext: vi.fn().mockResolvedValue({ data: { data: { current_node: null, scope_ids: [], breadcrumb: [] } } }),
}));

vi.mock('@/router/index.js', () => ({
  default: { push: vi.fn() },
}));

import AnalyticsPage from '@/pages/AnalyticsPage.vue';
import * as analyticsApi from '@/api/analytics.js';

const commonStubs = {
  AppButton: { template: '<button><slot /></button>' },
  AppInput: { template: '<input />' },
  AppBreadcrumb: { template: '<nav />' },
  AppLoadingState: { template: '<div class="loading" />' },
  AppErrorState: { template: '<div class="error"><button class="retry" @click="$emit(\'retry\')">Retry</button></div>', props: ['message', 'retryable'], emits: ['retry'] },
  AppEmptyState: { template: '<div class="empty" />', props: ['title', 'description'] },
  AppToast: {
    template: '<div class="toast" />',
    setup(_, { expose }) {
      expose({ addToast() {} });
    },
  },
};

describe('AnalyticsPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  function mountPage(options = {}) {
    return shallowMount(AnalyticsPage, {
      global: {
        stubs: commonStubs,
      },
      ...options,
    });
  }

  it('renders the page title', () => {
    const wrapper = mountPage();
    expect(wrapper.find('.page-header__title').text()).toBe('Analytics');
  });

  it('loads KPI data on mount', async () => {
    mountPage();
    await flushPromises();
    expect(analyticsApi.getKPIs).toHaveBeenCalledTimes(1);
  });

  it('shows empty state when no KPIs', async () => {
    analyticsApi.getKPIs.mockResolvedValueOnce({ data: { items: [] } });
    const wrapper = mountPage();
    await flushPromises();
    expect(wrapper.findComponent(commonStubs.AppEmptyState).exists()).toBe(true);
  });

  it('shows loading state while KPIs load', async () => {
    // Use a never-resolving promise to keep the loading state visible.
    analyticsApi.getKPIs.mockImplementation(() => new Promise(() => {}));
    const wrapper = mountPage();
    // Give the component a tick to start loading.
    await nextTick();
    // In loading state, skeleton tiles should be visible.
    const skeletons = wrapper.findAll('.kpi-tile--skeleton');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('shows error state on API failure', async () => {
    analyticsApi.getKPIs.mockRejectedValueOnce(new Error('Network error'));
    const wrapper = mountPage();
    await flushPromises();
    expect(wrapper.findComponent(commonStubs.AppErrorState).exists()).toBe(true);
  });

  it('renders KPI tiles when data is available', async () => {
    analyticsApi.getKPIs.mockResolvedValueOnce({
      data: [
        { key: 'kpi1', label: 'Total SKUs', value: 150, changePercent: 5.2 },
        { key: 'kpi2', label: 'Fill Rate', value: 92.4, changePercent: -1.3 },
      ],
    });
    const wrapper = mountPage();
    await flushPromises();
    const tiles = wrapper.findAll('.kpi-tile');
    expect(tiles.length).toBe(2);
  });

  it('renders date preset buttons', () => {
    const wrapper = mountPage();
    const dateButtons = wrapper.findAll('.date-btn');
    expect(dateButtons.length).toBeGreaterThanOrEqual(4);
  });

  it('renders scope chips', () => {
    const wrapper = mountPage();
    const chips = wrapper.findAll('.scope-chip');
    expect(chips.length).toBeGreaterThanOrEqual(1);
  });
});
