import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, shallowMount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

import AppButton from '@/components/common/AppButton.vue';
import AppChip from '@/components/common/AppChip.vue';
import AppDialog from '@/components/common/AppDialog.vue';
import AppTable from '@/components/common/AppTable.vue';
import AppInput from '@/components/common/AppInput.vue';
import AppFileUpload from '@/components/common/AppFileUpload.vue';

describe('AppButton', () => {
  it('renders slot content', () => {
    const wrapper = mount(AppButton, { slots: { default: 'Save' } });
    expect(wrapper.text()).toContain('Save');
  });

  it('is disabled when loading is true', () => {
    const wrapper = mount(AppButton, { props: { loading: true }, slots: { default: 'Save' } });
    expect(wrapper.find('button').element.disabled).toBe(true);
    expect(wrapper.classes()).toContain('app-btn--loading');
  });

  it('is disabled when disabled prop is true', () => {
    const wrapper = mount(AppButton, { props: { disabled: true }, slots: { default: 'Save' } });
    expect(wrapper.find('button').element.disabled).toBe(true);
  });

  it('shows spinner when loading', () => {
    const wrapper = mount(AppButton, { props: { loading: true }, slots: { default: 'Save' } });
    expect(wrapper.find('.app-btn__spinner').exists()).toBe(true);
    expect(wrapper.find('.app-btn__content--hidden').exists()).toBe(true);
  });

  it('applies variant class', () => {
    const wrapper = mount(AppButton, { props: { variant: 'danger' }, slots: { default: 'Delete' } });
    expect(wrapper.classes()).toContain('app-btn--danger');
  });

  it('applies size class', () => {
    const wrapper = mount(AppButton, { props: { size: 'lg' }, slots: { default: 'Big' } });
    expect(wrapper.classes()).toContain('app-btn--lg');
  });

  it('emits click event when clicked', async () => {
    const wrapper = mount(AppButton, { slots: { default: 'Click Me' } });
    await wrapper.find('button').trigger('click');
    expect(wrapper.emitted('click')).toHaveLength(1);
  });
});

describe('AppChip', () => {
  it.each([
    ['active', 'success'],
    ['running', 'info'],
    ['failed', 'danger'],
    ['blocked', 'warning'],
    ['awaiting-ack', 'warning'],
    ['ready', 'success'],
    ['inactive', 'neutral'],
  ])('maps status "%s" to variant "%s"', (status, expectedVariant) => {
    const wrapper = mount(AppChip, { props: { status, label: status } });
    expect(wrapper.classes()).toContain(`app-chip--${expectedVariant}`);
  });

  it('uses explicit variant over status mapping', () => {
    const wrapper = mount(AppChip, { props: { status: 'active', variant: 'danger', label: 'Override' } });
    expect(wrapper.classes()).toContain('app-chip--danger');
  });

  it('renders the label text', () => {
    const wrapper = mount(AppChip, { props: { label: 'Running', status: 'running' } });
    expect(wrapper.find('.app-chip__label').text()).toBe('Running');
  });
});

describe('AppDialog', () => {
  it('renders when modelValue is true', () => {
    const wrapper = mount(AppDialog, {
      props: { modelValue: true, title: 'Confirm' },
      slots: { default: '<p>Are you sure?</p>' },
      global: { stubs: { teleport: true } },
    });
    expect(wrapper.find('.dialog').exists()).toBe(true);
    expect(wrapper.find('.dialog__title').text()).toBe('Confirm');
  });

  it('does not render when modelValue is false', () => {
    const wrapper = mount(AppDialog, {
      props: { modelValue: false, title: 'Hidden' },
      global: { stubs: { teleport: true } },
    });
    expect(wrapper.find('.dialog').exists()).toBe(false);
  });

  it('emits close on close button click', async () => {
    const wrapper = mount(AppDialog, {
      props: { modelValue: true, title: 'Test' },
      global: { stubs: { teleport: true } },
    });
    await wrapper.find('.dialog__close').trigger('click');
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false]);
    expect(wrapper.emitted('close')).toHaveLength(1);
  });

  it('does not close on overlay click when persistent', async () => {
    const wrapper = mount(AppDialog, {
      props: { modelValue: true, title: 'Persistent', persistent: true },
      global: { stubs: { teleport: true } },
    });
    await wrapper.find('.dialog-overlay').trigger('click');
    expect(wrapper.emitted('update:modelValue')).toBeUndefined();
  });
});

describe('AppTable', () => {
  const columns = [
    { key: 'name', label: 'Name' },
    { key: 'status', label: 'Status' },
  ];

  it('renders column headers', () => {
    const wrapper = mount(AppTable, { props: { columns, rows: [] } });
    const headers = wrapper.findAll('th');
    expect(headers).toHaveLength(2);
    expect(headers[0].text()).toContain('Name');
  });

  it('shows empty state when rows is empty', () => {
    const wrapper = mount(AppTable, { props: { columns, rows: [] } });
    expect(wrapper.find('.app-table__empty').exists()).toBe(true);
    expect(wrapper.text()).toContain('No data available');
  });

  it('renders pagination when totalPages > 1', () => {
    const wrapper = mount(AppTable, {
      props: { columns, rows: [{ name: 'A', status: 'ok' }], currentPage: 1, totalPages: 3 },
    });
    expect(wrapper.find('.app-table__pagination').exists()).toBe(true);
    expect(wrapper.text()).toContain('Page 1 of 3');
  });

  it('emits page-change on next page click', async () => {
    const wrapper = mount(AppTable, {
      props: { columns, rows: [{ name: 'A', status: 'ok' }], currentPage: 1, totalPages: 3 },
    });
    const nextBtn = wrapper.findAll('.app-table__page-btn').at(-1);
    await nextBtn.trigger('click');
    expect(wrapper.emitted('page-change')?.[0]).toEqual([2]);
  });

  it('disables previous button on first page', () => {
    const wrapper = mount(AppTable, {
      props: { columns, rows: [{ name: 'A', status: 'ok' }], currentPage: 1, totalPages: 3 },
    });
    const prevBtn = wrapper.findAll('.app-table__page-btn').at(0);
    expect(prevBtn.element.disabled).toBe(true);
  });
});

describe('AppInput', () => {
  it('displays validation error message', () => {
    const wrapper = mount(AppInput, { props: { modelValue: '', error: 'Field is required' } });
    expect(wrapper.find('.app-input__error').text()).toBe('Field is required');
    expect(wrapper.classes()).toContain('app-input--error');
  });

  it('shows hint when no error', () => {
    const wrapper = mount(AppInput, { props: { modelValue: '', hint: 'Enter a value' } });
    expect(wrapper.find('.app-input__hint').text()).toBe('Enter a value');
    expect(wrapper.find('.app-input__error').exists()).toBe(false);
  });

  it('emits update:modelValue on input', async () => {
    const wrapper = mount(AppInput, { props: { modelValue: '' } });
    await wrapper.find('input').setValue('hello');
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['hello']);
  });
});

describe('AppFileUpload', () => {
  it('rejects files with disallowed extension', async () => {
    const wrapper = mount(AppFileUpload, { props: { accept: '.csv,.xlsx' } });
    const file = new File(['test'], 'data.exe', { type: 'application/x-msdownload' });
    const input = wrapper.find('input[type="file"]');
    Object.defineProperty(input.element, 'files', { value: [file] });
    await input.trigger('change');
    expect(wrapper.emitted('error')?.[0]?.[0]).toContain('File type not allowed');
  });

  it('rejects files exceeding maxSize', async () => {
    const wrapper = mount(AppFileUpload, { props: { maxSize: 1024 } });
    const bigContent = new Uint8Array(2048);
    const file = new File([bigContent], 'big.csv', { type: 'text/csv' });
    const input = wrapper.find('input[type="file"]');
    Object.defineProperty(input.element, 'files', { value: [file] });
    await input.trigger('change');
    expect(wrapper.emitted('error')?.[0]?.[0]).toContain('File too large');
  });

  it('accepts a valid file and emits file-selected', async () => {
    const wrapper = mount(AppFileUpload, {
      props: { accept: '.csv', maxSize: 10 * 1024 * 1024 },
    });
    const file = new File(['a,b,c'], 'data.csv', { type: 'text/csv' });
    const input = wrapper.find('input[type="file"]');
    Object.defineProperty(input.element, 'files', { value: [file] });
    await input.trigger('change');
    expect(wrapper.emitted('file-selected')?.[0]?.[0]).toBe(file);
    expect(wrapper.emitted('error')).toBeUndefined();
  });
});