import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn(),
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

const mockPush = vi.fn();
const mockRoute = { query: {} };

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useRoute: () => mockRoute,
}));

vi.mock('@/router/index.js', () => ({
  default: { push: vi.fn() },
}));

import LoginPage from '@/pages/LoginPage.vue';
import { useAuthStore } from '@/stores/auth.js';

function createWrapper() {
  return mount(LoginPage, {
    global: {
      plugins: [createPinia()],
    },
  });
}

describe('LoginPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    mockRoute.query = {};
  });

  it('validates that username is required', async () => {
    const wrapper = createWrapper();
    wrapper.vm.username = '';
    wrapper.vm.password = 'ValidPass1!xyz';

    await wrapper.find('form').trigger('submit');
    await nextTick();

    expect(wrapper.find('.form-field__error').text()).toContain('Username is required');
  });

  it('validates that password is required', async () => {
    const wrapper = createWrapper();
    wrapper.vm.username = 'admin';
    wrapper.vm.password = '';

    await wrapper.find('form').trigger('submit');
    await nextTick();

    const errors = wrapper.findAll('.form-field__error');
    const passwordError = errors.find(e => e.text().includes('Password is required'));
    expect(passwordError).toBeDefined();
  });

  it('rejects password shorter than 12 characters', async () => {
    const wrapper = createWrapper();
    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'Short1!ab';

    await wrapper.find('form').trigger('submit');
    await nextTick();

    const errors = wrapper.findAll('.form-field__error');
    const passwordError = errors.find(e => e.text().includes('at least 12 characters'));
    expect(passwordError).toBeDefined();
  });

  it('rejects password missing complexity requirements', async () => {
    const wrapper = createWrapper();
    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'alllowercase1';

    await wrapper.find('form').trigger('submit');
    await nextTick();

    const errors = wrapper.findAll('.form-field__error');
    const complexityError = errors.find(e => e.text().includes('uppercase'));
    expect(complexityError).toBeDefined();
  });

  it('accepts a password meeting all complexity rules', async () => {
    const wrapper = createWrapper();
    const authStore = useAuthStore();
    authStore.login = vi.fn().mockResolvedValue({});

    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'MyP@ssword123';

    await wrapper.find('form').trigger('submit');
    await flushPromises();

    const errors = wrapper.findAll('.form-field__error');
    expect(errors).toHaveLength(0);
    expect(authStore.login).toHaveBeenCalledWith('admin', 'MyP@ssword123');
  });

  it('shows locked account message on 423 response', async () => {
    const wrapper = createWrapper();
    const authStore = useAuthStore();
    authStore.login = vi.fn().mockRejectedValue({
      response: { status: 423, data: { lockedUntil: null } },
    });

    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'MyP@ssword123';

    await wrapper.find('form').trigger('submit');
    await flushPromises();

    expect(wrapper.find('.login-alert--locked').exists()).toBe(true);
    expect(wrapper.text()).toContain('Account is temporarily locked');
  });

  it('shows loading state and disables button during submit', async () => {
    const wrapper = createWrapper();
    let resolveLogin;
    const loginPromise = new Promise(r => { resolveLogin = r; });
    const authStore = useAuthStore();
    authStore.login = vi.fn().mockReturnValue(loginPromise);

    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'MyP@ssword123';

    await wrapper.find('form').trigger('submit');
    await nextTick();

    expect(wrapper.vm.loading).toBe(true);
    expect(wrapper.find('.login-btn').element.disabled).toBe(true);
    expect(wrapper.find('.login-btn__spinner').exists()).toBe(true);

    resolveLogin({});
    await flushPromises();

    expect(wrapper.vm.loading).toBe(false);
  });

  it('enter key on password field triggers form submit', async () => {
    const wrapper = createWrapper();
    const authStore = useAuthStore();
    authStore.login = vi.fn().mockResolvedValue({});

    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'MyP@ssword123';

    await wrapper.find('#login-password').trigger('keydown.enter');
    await flushPromises();

    expect(authStore.login).toHaveBeenCalledWith('admin', 'MyP@ssword123');
  });

  it('redirects to default route on successful login', async () => {
    const wrapper = createWrapper();
    const authStore = useAuthStore();
    authStore.login = vi.fn().mockResolvedValue({});
    mockRoute.query = {};

    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'MyP@ssword123';

    await wrapper.find('form').trigger('submit');
    await flushPromises();

    expect(mockPush).toHaveBeenCalledWith('/master/sku');
  });

  it('redirects to query param route when redirect is set', async () => {
    const wrapper = createWrapper();
    const authStore = useAuthStore();
    authStore.login = vi.fn().mockResolvedValue({});
    mockRoute.query = { redirect: '/ingestion' };

    wrapper.vm.username = 'admin';
    wrapper.vm.password = 'MyP@ssword123';

    await wrapper.find('form').trigger('submit');
    await flushPromises();

    expect(mockPush).toHaveBeenCalledWith('/ingestion');
  });
});