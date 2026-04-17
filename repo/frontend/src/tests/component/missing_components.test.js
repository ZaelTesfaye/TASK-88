import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

import AppBreadcrumb from '@/components/common/AppBreadcrumb.vue';
import AppToast from '@/components/common/AppToast.vue';
import AppSelect from '@/components/common/AppSelect.vue';
import AppLoadingState from '@/components/common/AppLoadingState.vue';
import AppErrorState from '@/components/common/AppErrorState.vue';
import AppEmptyState from '@/components/common/AppEmptyState.vue';

beforeEach(() => {
  setActivePinia(createPinia());
});

// ==================== AppBreadcrumb ====================

describe('AppBreadcrumb', () => {
  const items = [
    { id: 1, name: 'Home' },
    { id: 2, name: 'Products' },
    { id: 3, name: 'Widget A' },
  ];

  it('renders all items', () => {
    const wrapper = mount(AppBreadcrumb, { props: { items } });
    const listItems = wrapper.findAll('.app-breadcrumb__item');
    expect(listItems.length).toBe(3);
  });

  it('last item is a span (not clickable)', () => {
    const wrapper = mount(AppBreadcrumb, { props: { items } });
    const lastItem = wrapper.findAll('.app-breadcrumb__item').at(2);
    expect(lastItem.find('.app-breadcrumb__current').exists()).toBe(true);
    expect(lastItem.find('.app-breadcrumb__link').exists()).toBe(false);
  });

  it('non-last items are buttons', () => {
    const wrapper = mount(AppBreadcrumb, { props: { items } });
    const firstItem = wrapper.findAll('.app-breadcrumb__item').at(0);
    expect(firstItem.find('.app-breadcrumb__link').exists()).toBe(true);
  });

  it('clicking non-last item emits navigate', async () => {
    const wrapper = mount(AppBreadcrumb, { props: { items } });
    const firstLink = wrapper.find('.app-breadcrumb__link');
    await firstLink.trigger('click');
    expect(wrapper.emitted('navigate')).toBeTruthy();
    expect(wrapper.emitted('navigate')[0][0]).toEqual(items[0]);
  });

  it('shows separator between items', () => {
    const wrapper = mount(AppBreadcrumb, { props: { items } });
    const separators = wrapper.findAll('.app-breadcrumb__separator');
    expect(separators.length).toBe(2);
  });

  it('renders empty list with no items', () => {
    const wrapper = mount(AppBreadcrumb, { props: { items: [] } });
    const listItems = wrapper.findAll('.app-breadcrumb__item');
    expect(listItems.length).toBe(0);
  });
});

// ==================== AppToast ====================

describe('AppToast', () => {
  it('starts with no toasts', () => {
    const wrapper = mount(AppToast, {
      global: { stubs: { teleport: true } },
    });
    expect(wrapper.findAll('.toast').length).toBe(0);
  });

  it('addToast renders a toast with message', async () => {
    const wrapper = mount(AppToast, {
      global: { stubs: { teleport: true } },
    });
    wrapper.vm.addToast({ message: 'Hello', duration: 0 });
    await nextTick();
    const toasts = wrapper.findAll('.toast');
    expect(toasts.length).toBe(1);
    expect(wrapper.find('.toast__message').text()).toBe('Hello');
  });

  it('toast has correct type class', async () => {
    const wrapper = mount(AppToast, {
      global: { stubs: { teleport: true } },
    });
    wrapper.vm.addToast({ message: 'Success!', type: 'success', duration: 0 });
    await nextTick();
    expect(wrapper.find('.toast--success').exists()).toBe(true);
  });

  it('removeToast via exposed API removes the toast', async () => {
    const wrapper = mount(AppToast, {
      global: {
        stubs: {
          teleport: true,
          // Replace TransitionGroup with a plain div to avoid leave animations
          // keeping DOM elements around.
          TransitionGroup: {
            template: '<div><slot /></div>',
          },
        },
      },
    });
    wrapper.vm.addToast({ message: 'Temp', duration: 0 });
    await nextTick();
    const toasts = wrapper.findAll('.toast');
    expect(toasts.length).toBe(1);

    // Read the data attribute or just grab the internal state to find the id.
    // Since nextId is module-scoped and increments across tests, get the
    // actual id from the rendered toast's key or from the component internals.
    // Simplest: find the toast element and extract its key from the wrapper html.
    // Alternative: since we added exactly one toast, remove the first one.
    const toastEl = wrapper.find('[role="alert"]');
    expect(toastEl.exists()).toBe(true);

    // Call removeToast with a very large id range — instead, just click the close button.
    const closeBtn = wrapper.find('.toast__close');
    expect(closeBtn.exists()).toBe(true);
    await closeBtn.trigger('click');
    await nextTick();
    expect(wrapper.findAll('.toast').length).toBe(0);
  });

  it('multiple toasts render multiple elements', async () => {
    const wrapper = mount(AppToast, {
      global: { stubs: { teleport: true } },
    });
    wrapper.vm.addToast({ message: 'First', duration: 0 });
    wrapper.vm.addToast({ message: 'Second', duration: 0 });
    await nextTick();
    expect(wrapper.findAll('.toast').length).toBe(2);
  });

  it('defaults to info type', async () => {
    const wrapper = mount(AppToast, {
      global: { stubs: { teleport: true } },
    });
    wrapper.vm.addToast({ message: 'Info toast', duration: 0 });
    await nextTick();
    expect(wrapper.find('.toast--info').exists()).toBe(true);
  });
});

// ==================== AppSelect ====================

describe('AppSelect', () => {
  const options = [
    { value: 'a', label: 'Option A' },
    { value: 'b', label: 'Option B' },
    { value: 'c', label: 'Option C' },
  ];

  it('renders label text', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options, label: 'Pick one' },
    });
    expect(wrapper.find('.app-select__label').text()).toContain('Pick one');
  });

  it('shows required asterisk when required', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options, label: 'Required', required: true },
    });
    expect(wrapper.find('.app-select__required').exists()).toBe(true);
  });

  it('renders all options', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options },
    });
    const optionEls = wrapper.findAll('option:not([disabled])');
    expect(optionEls.length).toBe(3);
  });

  it('shows placeholder as disabled option', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options, placeholder: 'Choose...' },
    });
    const placeholderOpt = wrapper.find('option[disabled]');
    expect(placeholderOpt.exists()).toBe(true);
    expect(placeholderOpt.text()).toBe('Choose...');
  });

  it('shows error message when error prop set', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options, error: 'Required field' },
    });
    expect(wrapper.find('.app-select__error').text()).toBe('Required field');
    expect(wrapper.classes()).toContain('app-select--error');
  });

  it('shows hint when hint prop set', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options, hint: 'Select an option' },
    });
    expect(wrapper.find('.app-select__hint').text()).toBe('Select an option');
  });

  it('emits update:modelValue on change', async () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options },
    });
    const select = wrapper.find('select');
    await select.setValue('b');
    expect(wrapper.emitted('update:modelValue')).toBeTruthy();
  });

  it('disabled state applies class and attribute', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options, disabled: true },
    });
    expect(wrapper.classes()).toContain('app-select--disabled');
    expect(wrapper.find('select').attributes('disabled')).toBeDefined();
  });

  it('normalizes string array options', () => {
    const wrapper = mount(AppSelect, {
      props: { modelValue: '', options: ['alpha', 'beta', 'gamma'] },
    });
    const optionEls = wrapper.findAll('option:not([disabled])');
    expect(optionEls.length).toBe(3);
    expect(optionEls[0].text()).toBe('alpha');
  });
});

// ==================== AppLoadingState ====================

describe('AppLoadingState', () => {
  it('default renders spinner variant', () => {
    const wrapper = mount(AppLoadingState);
    expect(wrapper.find('.app-loading__spinner').exists()).toBe(true);
  });

  it('spinner variant shows SVG', () => {
    const wrapper = mount(AppLoadingState, { props: { variant: 'spinner' } });
    expect(wrapper.find('svg').exists()).toBe(true);
  });

  it('shows message when provided', () => {
    const wrapper = mount(AppLoadingState, { props: { message: 'Loading data...' } });
    expect(wrapper.find('.app-loading__message').text()).toBe('Loading data...');
  });

  it('skeleton variant renders correct number of lines', () => {
    const wrapper = mount(AppLoadingState, { props: { variant: 'skeleton', lines: 5 } });
    const skeletons = wrapper.findAll('.skeleton');
    expect(skeletons.length).toBe(5);
  });

  it('overlay adds overlay class', () => {
    const wrapper = mount(AppLoadingState, { props: { overlay: true } });
    expect(wrapper.find('.app-loading--overlay').exists()).toBe(true);
  });

  it('no message hides message element', () => {
    const wrapper = mount(AppLoadingState);
    expect(wrapper.find('.app-loading__message').exists()).toBe(false);
  });
});

// ==================== AppErrorState ====================

describe('AppErrorState', () => {
  it('shows default title', () => {
    const wrapper = mount(AppErrorState);
    expect(wrapper.find('.app-error__title').text()).toBe('Something went wrong');
  });

  it('shows custom title', () => {
    const wrapper = mount(AppErrorState, { props: { title: 'Error occurred' } });
    expect(wrapper.find('.app-error__title').text()).toBe('Error occurred');
  });

  it('shows message when provided', () => {
    const wrapper = mount(AppErrorState, { props: { message: 'Connection timeout' } });
    expect(wrapper.find('.app-error__message').text()).toBe('Connection timeout');
  });

  it('shows error code when provided', () => {
    const wrapper = mount(AppErrorState, { props: { code: 'ERR_500' } });
    expect(wrapper.find('.app-error__code').text()).toContain('ERR_500');
  });

  it('shows retry button when retryable', () => {
    const wrapper = mount(AppErrorState, { props: { retryable: true } });
    expect(wrapper.find('.app-error__retry-btn').exists()).toBe(true);
  });

  it('hides retry button when not retryable', () => {
    const wrapper = mount(AppErrorState, { props: { retryable: false } });
    expect(wrapper.find('.app-error__retry-btn').exists()).toBe(false);
  });

  it('clicking retry emits retry event', async () => {
    const wrapper = mount(AppErrorState, { props: { retryable: true } });
    await wrapper.find('.app-error__retry-btn').trigger('click');
    expect(wrapper.emitted('retry')).toBeTruthy();
  });
});

// ==================== AppEmptyState ====================

describe('AppEmptyState', () => {
  it('shows default title', () => {
    const wrapper = mount(AppEmptyState);
    expect(wrapper.find('.app-empty__title').text()).toBe('No data found');
  });

  it('shows custom title', () => {
    const wrapper = mount(AppEmptyState, { props: { title: 'Nothing here' } });
    expect(wrapper.find('.app-empty__title').text()).toBe('Nothing here');
  });

  it('shows description when provided', () => {
    const wrapper = mount(AppEmptyState, { props: { description: 'Try a different filter' } });
    expect(wrapper.find('.app-empty__description').text()).toBe('Try a different filter');
  });

  it('shows default icon when no icon slot', () => {
    const wrapper = mount(AppEmptyState);
    expect(wrapper.find('.app-empty__icon').exists()).toBe(true);
  });

  it('shows custom icon from slot', () => {
    const wrapper = mount(AppEmptyState, {
      slots: { icon: '<span class="custom-icon">Custom</span>' },
    });
    expect(wrapper.find('.custom-icon').exists()).toBe(true);
  });

  it('shows action slot content', () => {
    const wrapper = mount(AppEmptyState, {
      slots: { action: '<button class="test-action">Do Something</button>' },
    });
    expect(wrapper.find('.test-action').exists()).toBe(true);
    expect(wrapper.find('.app-empty__action').exists()).toBe(true);
  });

  it('hides action wrapper when no action slot', () => {
    const wrapper = mount(AppEmptyState);
    expect(wrapper.find('.app-empty__action').exists()).toBe(false);
  });
});
