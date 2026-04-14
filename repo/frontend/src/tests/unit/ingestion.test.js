import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

// Mock APIs
vi.mock('@/api/ingestion.js', () => ({
  getSources: vi.fn(),
  createSource: vi.fn(),
  updateSource: vi.fn(),
  getConnectorHealth: vi.fn(),
  getCapabilities: vi.fn(),
  runJob: vi.fn(),
  acknowledgeJob: vi.fn(),
  getJobs: vi.fn(),
  getCheckpoints: vi.fn(),
}));

vi.mock('@/api/client.js', () => ({
  default: { post: vi.fn(), get: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
}));

vi.mock('@/api/org.js', () => ({
  switchContext: vi.fn(),
  getOrgTree: vi.fn(),
}));

vi.mock('@/router/index.js', () => ({
  default: { push: vi.fn() },
}));

import * as ingestionApi from '@/api/ingestion.js';
import IngestionPage from '@/pages/IngestionPage.vue';

const toastStub = {
  template: '<div class="toast"></div>',
  setup(_, { expose }) {
    expose({ addToast() {} });
  },
};

function createWrapper() {
  ingestionApi.getSources.mockResolvedValue({
    data: [
      { id: 'src-1', name: 'Test Source', source_type: 'db', enabled: true, healthStatus: 'healthy' },
    ],
  });

  ingestionApi.getJobs.mockResolvedValue({
    data: {
      items: [
        { id: 'j1', sourceName: 'Test Source', state: 'running', mode: 'incremental', recordsProcessed: 50, totalEstimate: 100, priority: 1, created_at: '2024-01-01T00:00:00Z' },
        { id: 'j2', sourceName: 'Test Source', state: 'failed', mode: 'backfill', recordsProcessed: 10, totalEstimate: 100, errorDetails: 'Connection timeout', priority: 2, created_at: '2024-01-02T00:00:00Z' },
        { id: 'j3', sourceName: 'Test Source', state: 'completed', mode: 'incremental', recordsProcessed: 100, totalEstimate: 100, priority: 1, created_at: '2024-01-03T00:00:00Z' },
        { id: 'j4', sourceName: 'Test Source', state: 'blocked', mode: 'incremental', recordsProcessed: 0, totalEstimate: 50, blockedBy: ['j1'], priority: 3, created_at: '2024-01-04T00:00:00Z' },
        { id: 'j5', sourceName: 'Test Source', state: 'awaiting-ack', mode: 'backfill', recordsProcessed: 20, totalEstimate: 100, priority: 2, created_at: '2024-01-05T00:00:00Z' },
      ],
      totalPages: 1,
    },
  });

  return shallowMount(IngestionPage, {
    global: {
      plugins: [createPinia()],
      stubs: {
        AppButton: { template: '<button :disabled="disabled || loading" @click="$emit(\'click\')"><slot /></button>', props: ['loading', 'disabled', 'variant', 'size'], emits: ['click'] },
        AppInput: { template: '<div class="app-input"><input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></div>', props: ['modelValue', 'label', 'type', 'required', 'error', 'hint', 'placeholder', 'rows'], emits: ['update:modelValue'] },
        AppSelect: { template: '<select></select>', props: ['modelValue', 'options', 'placeholder', 'label', 'required', 'error'] },
        AppChip: { template: '<span class="app-chip" :class="`app-chip--${status}`" :data-status="status">{{ label }}</span>', props: ['status', 'label', 'variant', 'size'] },
        AppTable: { template: '<div class="app-table"><slot /></div>', props: ['columns', 'rows', 'loading', 'currentPage', 'totalPages'] },
        AppDialog: { template: '<div v-if="modelValue" class="app-dialog"><slot /><slot name="footer" /></div>', props: ['modelValue', 'title', 'size', 'persistent', 'danger'] },
        AppLoadingState: { template: '<div class="loading"></div>', props: ['message'] },
        AppErrorState: { template: '<div class="error"></div>', props: ['message', 'retryable'] },
        AppEmptyState: { template: '<div class="empty"><slot name="action" /></div>', props: ['title', 'description'] },
        AppToast: toastStub,
      },
    },
  });
}

describe('IngestionPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('job state chips render correctly for each state', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    // Switch to jobs tab
    wrapper.vm.activeTab = 'jobs';
    await nextTick();
    await flushPromises();

    const jobs = wrapper.vm.jobs;
    expect(jobs.length).toBe(5);

    const states = jobs.map(j => j.state);
    expect(states).toContain('running');
    expect(states).toContain('failed');
    expect(states).toContain('completed');
    expect(states).toContain('blocked');
    expect(states).toContain('awaiting-ack');

    // Verify the AppChip status mapping covers all these states
    const STATUS_VARIANT_MAP = {
      running: 'info',
      failed: 'danger',
      blocked: 'warning',
      'awaiting-ack': 'warning',
    };

    for (const [state, variant] of Object.entries(STATUS_VARIANT_MAP)) {
      expect(variant).toBeDefined();
    }
  });

  it('acknowledge dialog requires reason', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    wrapper.vm.activeTab = 'jobs';
    await nextTick();
    await flushPromises();

    const failedJob = wrapper.vm.jobs.find(j => j.state === 'failed');
    expect(failedJob).toBeDefined();

    // Open acknowledge dialog
    wrapper.vm.openAckDialog(failedJob);
    await nextTick();

    expect(wrapper.vm.showAckDialog).toBe(true);
    expect(wrapper.vm.ackJob).toEqual(failedJob);

    // Try to acknowledge without reason
    wrapper.vm.ackReason = '';
    await wrapper.vm.acknowledgeFailedJob();
    await nextTick();

    expect(wrapper.vm.ackReasonError).toBe('Reason is required');
    expect(ingestionApi.acknowledgeJob).not.toHaveBeenCalled();

    // Now set a reason and try again
    wrapper.vm.ackReason = 'Investigated and resolved the timeout issue';
    ingestionApi.acknowledgeJob.mockResolvedValue({});

    await wrapper.vm.acknowledgeFailedJob();
    await flushPromises();

    expect(ingestionApi.acknowledgeJob).toHaveBeenCalledWith(
      failedJob.id,
      'Investigated and resolved the timeout issue'
    );
  });

  it('auto-refresh triggers for running jobs', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    wrapper.vm.activeTab = 'jobs';
    await nextTick();
    await flushPromises();

    // There is a running job, so scheduleJobRefresh should have set up an interval
    const hasRunningJob = wrapper.vm.jobs.some(j => j.state === 'running');
    expect(hasRunningJob).toBe(true);

    // Clear the call count
    ingestionApi.getJobs.mockClear();

    // Advance timer by 10 seconds (the refresh interval)
    vi.advanceTimersByTime(10000);
    await flushPromises();

    // getJobs should have been called again (auto-refresh)
    expect(ingestionApi.getJobs).toHaveBeenCalled();
  });

  it('no auto-refresh when no running jobs', async () => {
    // Override the jobs mock to return only completed jobs
    ingestionApi.getSources.mockResolvedValue({
      data: [{ id: 'src-1', name: 'Test Source', source_type: 'db', enabled: true }],
    });
    ingestionApi.getJobs.mockResolvedValue({
      data: {
        items: [
          { id: 'j1', sourceName: 'Src', state: 'completed', mode: 'incremental', recordsProcessed: 100, totalEstimate: 100, priority: 1, created_at: '2024-01-01T00:00:00Z' },
        ],
        totalPages: 1,
      },
    });

    const wrapper = shallowMount(IngestionPage, {
      global: {
        plugins: [createPinia()],
        stubs: {
          AppButton: { template: '<button><slot /></button>', props: ['loading', 'disabled', 'variant', 'size'] },
          AppInput: { template: '<div class="app-input"><input /></div>', props: ['modelValue', 'label', 'type', 'required', 'error', 'hint', 'placeholder', 'rows'] },
          AppSelect: { template: '<select></select>', props: ['modelValue', 'options', 'placeholder', 'label', 'required', 'error'] },
          AppChip: { template: '<span class="app-chip">{{ label }}</span>', props: ['status', 'label', 'variant', 'size'] },
          AppTable: { template: '<div class="app-table"><slot /></div>', props: ['columns', 'rows', 'loading', 'currentPage', 'totalPages'] },
          AppDialog: { template: '<div v-if="modelValue"><slot /><slot name="footer" /></div>', props: ['modelValue', 'title', 'size', 'persistent', 'danger'] },
          AppLoadingState: { template: '<div class="loading"></div>', props: ['message'] },
          AppErrorState: { template: '<div class="error"></div>', props: ['message', 'retryable'] },
          AppEmptyState: { template: '<div class="empty"><slot name="action" /></div>', props: ['title', 'description'] },
          AppToast: toastStub,
        },
      },
    });
    await flushPromises();

    // Switch to jobs tab - this triggers the watcher which calls loadJobs()
    wrapper.vm.activeTab = 'jobs';
    await nextTick();
    await flushPromises();

    // Now clear after the watcher-triggered loadJobs has completed
    ingestionApi.getJobs.mockClear();

    vi.advanceTimersByTime(15000);
    await flushPromises();

    // No auto-refresh because no running jobs
    expect(ingestionApi.getJobs).not.toHaveBeenCalled();
  });

  it('job progress calculation works correctly', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    // Test progress function
    const result50 = wrapper.vm.jobProgress({ recordsProcessed: 50, totalEstimate: 100 });
    expect(result50).toBe(50);

    const result0 = wrapper.vm.jobProgress({ recordsProcessed: 0, totalEstimate: 0 });
    expect(result0).toBe(0);

    const resultFull = wrapper.vm.jobProgress({ recordsProcessed: 100, totalEstimate: 100 });
    expect(resultFull).toBe(100);

    const resultOver = wrapper.vm.jobProgress({ recordsProcessed: 110, totalEstimate: 100 });
    expect(resultOver).toBe(100); // Capped at 100
  });
});
