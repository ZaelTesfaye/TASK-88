import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

// Mock APIs
vi.mock('@/api/security.js', () => ({
  getSensitiveFields: vi.fn(),
  updateSensitiveFields: vi.fn(),
  createPasswordResetRequest: vi.fn(),
  approvePasswordResetRequest: vi.fn(),
  getRetentionPolicies: vi.fn(),
  updateRetentionPolicies: vi.fn(),
  createLegalHold: vi.fn(),
  dryRunPurge: vi.fn(),
  executePurge: vi.fn(),
}));

vi.mock('@/api/audit.js', () => ({
  getLogs: vi.fn(),
  createDeleteRequest: vi.fn(),
  approveDeleteRequest: vi.fn(),
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

import * as securityApi from '@/api/security.js';
import * as auditApi from '@/api/audit.js';
import SecurityAdminPage from '@/pages/SecurityAdminPage.vue';
import { useAuthStore } from '@/stores/auth.js';

const toastStub = {
  template: '<div class="toast"></div>',
  setup(_, { expose }) {
    expose({ addToast() {} });
  },
};

const commonStubs = {
  AppButton: { template: '<button :disabled="disabled || loading" @click="$emit(\'click\')"><slot /></button>', props: ['loading', 'disabled', 'variant', 'size'], emits: ['click'] },
  AppInput: { template: '<div class="app-input"><input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></div>', props: ['modelValue', 'label', 'type', 'required', 'error', 'hint', 'placeholder', 'disabled', 'rows'], emits: ['update:modelValue'] },
  AppSelect: { template: '<select></select>', props: ['modelValue', 'options', 'placeholder', 'label'] },
  AppChip: { template: '<span class="app-chip" :class="`app-chip--${status}`">{{ label }}</span>', props: ['status', 'label', 'variant', 'size'] },
  AppTable: { template: '<div class="app-table"><slot /></div>', props: ['columns', 'rows', 'loading', 'currentPage', 'totalPages'] },
  AppDialog: { template: '<div v-if="modelValue" class="app-dialog"><slot /><slot name="footer" /></div>', props: ['modelValue', 'title', 'size', 'persistent', 'danger'] },
  AppLoadingState: { template: '<div class="loading"></div>', props: ['message'] },
  AppErrorState: { template: '<div class="error"></div>', props: ['message', 'retryable'] },
  AppEmptyState: { template: '<div class="empty"><slot name="action" /></div>', props: ['title', 'description'] },
  AppToast: toastStub,
};

function createWrapper(pinia) {
  return shallowMount(SecurityAdminPage, {
    global: {
      plugins: [pinia || createPinia()],
      stubs: commonStubs,
    },
  });
}

describe('SecurityAdminPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  describe('Password Reset Approval Flow', () => {
    it('password reset approval flow visibility', async () => {
      const pinia = createPinia();
      setActivePinia(pinia);

      // Mock: on mount, loadFields calls getSensitiveFields() -> returns fields data
      // When passwords tab is selected, loadPasswordResets calls getSensitiveFields({ type: 'password_resets' })
      securityApi.getSensitiveFields.mockImplementation((params) => {
        if (params?.type === 'password_resets') {
          return Promise.resolve({
            data: [
              { id: 'pr1', user: 'john', requested_by: 'admin', state: 'pending', created_at: '2024-01-01T00:00:00Z' },
              { id: 'pr2', user: 'jane', requested_by: 'admin', state: 'approved', created_at: '2024-01-02T00:00:00Z' },
            ],
          });
        }
        return Promise.resolve({ data: [] });
      });

      const wrapper = createWrapper(pinia);
      await flushPromises();

      // Switch to passwords tab
      wrapper.vm.activeTab = 'passwords';
      await nextTick();
      await flushPromises();

      expect(wrapper.vm.passwordResets.length).toBe(2);
      const pendingReset = wrapper.vm.passwordResets.find(r => r.state === 'pending');
      const approvedReset = wrapper.vm.passwordResets.find(r => r.state === 'approved');

      expect(pendingReset).toBeDefined();
      expect(approvedReset).toBeDefined();

      // Pending resets should show approve button (state === 'pending')
      expect(pendingReset.state).toBe('pending');
      // Approved resets should not show approve button
      expect(approvedReset.state).toBe('approved');
    });
  });

  describe('Dual-Approval Progress Indicator', () => {
    it('dual-approval progress indicator (0/2, 1/2, 2/2)', async () => {
      const pinia = createPinia();
      setActivePinia(pinia);

      securityApi.getSensitiveFields.mockResolvedValue({ data: [] });
      auditApi.getLogs.mockResolvedValue({
        data: [
          { id: 'ar1', requested_by: 'admin', reason: 'GDPR request', state: 'pending', approvals: 0, approvedBy: [] },
          { id: 'ar2', requested_by: 'admin', reason: 'User request', state: 'pending', approvals: 1, approvedBy: ['user-1'] },
          { id: 'ar3', requested_by: 'admin', reason: 'Compliance', state: 'approved', approvals: 2, approvedBy: ['user-1', 'user-2'] },
        ],
      });

      const wrapper = createWrapper(pinia);
      await flushPromises();

      // Switch to audit tab
      wrapper.vm.activeTab = 'audit';
      await nextTick();
      await flushPromises();

      const requests = wrapper.vm.auditRequests;
      expect(requests.length).toBe(3);

      // 0/2 approved
      expect(requests[0].approvals).toBe(0);
      // 1/2 approved
      expect(requests[1].approvals).toBe(1);
      // 2/2 approved
      expect(requests[2].approvals).toBe(2);

      expect(requests[0].approvals >= 1).toBe(false);
      expect(requests[0].approvals >= 2).toBe(false);
      expect(requests[1].approvals >= 1).toBe(true);
      expect(requests[1].approvals >= 2).toBe(false);
      expect(requests[2].approvals >= 1).toBe(true);
      expect(requests[2].approvals >= 2).toBe(true);
    });
  });

  describe('Approve Button Disabled If User Already Approved', () => {
    it('approve button disabled if user already approved', async () => {
      const pinia = createPinia();
      setActivePinia(pinia);

      // Set up auth store with known user BEFORE creating the component
      const authStore = useAuthStore();
      authStore.token = 'test-token';
      authStore.user = { id: 'current-user-id', username: 'admin', role: 'system_admin' };

      securityApi.getSensitiveFields.mockResolvedValue({ data: [] });
      auditApi.getLogs.mockResolvedValue({
        data: [
          {
            id: 'ar1',
            requested_by: 'other-admin',
            reason: 'GDPR request',
            state: 'pending',
            approvals: 1,
            approvedBy: ['current-user-id'],
          },
          {
            id: 'ar2',
            requested_by: 'other-admin',
            reason: 'Another request',
            state: 'pending',
            approvals: 0,
            approvedBy: [],
          },
        ],
      });

      const wrapper = createWrapper(pinia);
      await flushPromises();

      wrapper.vm.activeTab = 'audit';
      await nextTick();
      await flushPromises();

      const requests = wrapper.vm.auditRequests;
      expect(requests.length).toBe(2);

      // hasUserApproved checks if authStore.user.id is in approvedBy
      const alreadyApproved = wrapper.vm.hasUserApproved(requests[0]);
      const notApproved = wrapper.vm.hasUserApproved(requests[1]);

      expect(alreadyApproved).toBe(true);
      expect(notApproved).toBe(false);
    });
  });

  it('password reset approval generates one-time token', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);

    securityApi.getSensitiveFields.mockImplementation((params) => {
      if (params?.type === 'password_resets') {
        return Promise.resolve({
          data: [
            { id: 'pr1', user: 'john', requested_by: 'admin', state: 'pending', created_at: '2024-01-01T00:00:00Z' },
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });

    securityApi.approvePasswordResetRequest.mockResolvedValue({
      data: { token: 'one-time-secret-token-abc123' },
    });

    const wrapper = createWrapper(pinia);
    await flushPromises();

    wrapper.vm.activeTab = 'passwords';
    await nextTick();
    await flushPromises();

    const pendingReset = wrapper.vm.passwordResets[0];
    expect(pendingReset._token).toBeNull();

    await wrapper.vm.approvePasswordReset(pendingReset);
    await flushPromises();

    expect(securityApi.approvePasswordResetRequest).toHaveBeenCalledWith('pr1');
    expect(pendingReset._token).toBe('one-time-secret-token-abc123');
    expect(pendingReset.state).toBe('approved');
  });
});
