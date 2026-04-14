import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

vi.mock('@/api/org.js', () => ({
  getOrgTree: vi.fn(),
  createNode: vi.fn(),
  updateNode: vi.fn(),
  deleteNode: vi.fn(),
  switchContext: vi.fn(),
}));

vi.mock('@/api/client.js', () => ({
  default: { post: vi.fn(), get: vi.fn(), put: vi.fn(), delete: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
}));

vi.mock('@/router/index.js', () => ({
  default: { push: vi.fn() },
}));

import * as orgApi from '@/api/org.js';
import OrgTreePage from '@/pages/OrgTreePage.vue';

const TREE_DATA = [
  {
    id: 'root-1',
    name: 'Global Corp',
    level_code: 'L1',
    level_label: 'Enterprise',
    parent_id: null,
    is_active: true,
    children: [
      {
        id: 'child-1',
        name: 'North America',
        level_code: 'L2',
        level_label: 'Region',
        parent_id: 'root-1',
        is_active: true,
        children: [
          { id: 'child-1a', name: 'US Operations', level_code: 'L3', level_label: 'Division', parent_id: 'child-1', is_active: true, children: [] },
        ],
      },
      {
        id: 'child-2',
        name: 'Europe',
        level_code: 'L2',
        level_label: 'Region',
        parent_id: 'root-1',
        is_active: false,
        children: [],
      },
    ],
  },
];

function createWrapper() {
  orgApi.getOrgTree.mockResolvedValue({ data: TREE_DATA });

  return shallowMount(OrgTreePage, {
    global: {
      plugins: [createPinia()],
      stubs: {
        AppButton: { template: '<button @click="$emit(\x27click\x27)"><slot /></button>', props: ['loading', 'disabled', 'variant'], emits: ['click'] },
        AppInput: { template: '<div class="app-input"><input :value="modelValue" @input="$emit(\x27update:modelValue\x27, $event.target.value)" /></div>', props: ['modelValue', 'label', 'required', 'error', 'placeholder'], emits: ['update:modelValue'] },
        AppSelect: { template: '<select></select>', props: ['modelValue', 'options', 'label', 'placeholder'] },
        AppDialog: { template: '<div v-if="modelValue" class="app-dialog"><slot /><slot name="footer" /></div>', props: ['modelValue', 'title', 'size', 'persistent', 'danger'] },
        AppChip: { template: '<span class="app-chip"></span>', props: ['status', 'label', 'size'] },
        AppLoadingState: { template: '<div class="loading"></div>', props: ['message'] },
        AppErrorState: { template: '<div class="error"></div>', props: ['message'] },
        AppEmptyState: { template: '<div class="empty"><slot name="action" /></div>', props: ['title', 'description'] },
      },
    },
  });
}

describe('OrgTreePage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('loads and renders the tree on mount', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    expect(orgApi.getOrgTree).toHaveBeenCalledTimes(1);
    expect(wrapper.vm.treeData).toHaveLength(1);
    expect(wrapper.vm.treeData[0].name).toBe('Global Corp');
  });

  describe('Expand / Collapse', () => {
    it('toggleExpand adds node id to expandedIds', async () => {
      const wrapper = createWrapper();
      await flushPromises();

      expect(wrapper.vm.expandedIds.has('root-1')).toBe(false);
      wrapper.vm.toggleExpand('root-1');
      await nextTick();
      expect(wrapper.vm.expandedIds.has('root-1')).toBe(true);
    });

    it('toggleExpand removes node id when already expanded', async () => {
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.toggleExpand('root-1');
      await nextTick();
      expect(wrapper.vm.expandedIds.has('root-1')).toBe(true);

      wrapper.vm.toggleExpand('root-1');
      await nextTick();
      expect(wrapper.vm.expandedIds.has('root-1')).toBe(false);
    });
  });

  describe('Create Node Validation', () => {
    it('validates that name, level_code, and level_label are required', async () => {
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.openCreateDialog(null);
      await nextTick();

      expect(wrapper.vm.showCreateDialog).toBe(true);

      wrapper.vm.createForm.name = '';
      wrapper.vm.createForm.level_code = '';
      wrapper.vm.createForm.level_label = '';

      await wrapper.vm.createNode();
      await nextTick();

      expect(wrapper.vm.createErrors.name).toBe('Required');
      expect(wrapper.vm.createErrors.level_code).toBe('Required');
      expect(wrapper.vm.createErrors.level_label).toBe('Required');
      expect(orgApi.createNode).not.toHaveBeenCalled();
    });

    it('calls API when all required fields are filled', async () => {
      orgApi.createNode.mockResolvedValue({});
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.openCreateDialog(TREE_DATA[0]);
      await nextTick();

      wrapper.vm.createForm.name = 'Asia Pacific';
      wrapper.vm.createForm.level_code = 'L2';
      wrapper.vm.createForm.level_label = 'Region';

      await wrapper.vm.createNode();
      await flushPromises();

      expect(orgApi.createNode).toHaveBeenCalledWith('root-1', expect.objectContaining({ name: 'Asia Pacific' }));
      expect(wrapper.vm.showCreateDialog).toBe(false);
    });
  });

  describe('Parent Dropdown Cycle Prevention', () => {
    it('excludes the selected node and its descendants from parent options', async () => {
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.selectNode(TREE_DATA[0]);
      await nextTick();

      const options = wrapper.vm.parentOptions;
      const optionValues = options.map(o => o.value);

      expect(optionValues).not.toContain('root-1');
      expect(optionValues).not.toContain('child-1');
      expect(optionValues).not.toContain('child-1a');
      expect(optionValues).not.toContain('child-2');
      expect(optionValues).toContain('');
    });

    it('allows selecting unrelated nodes as parent', async () => {
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.selectNode(TREE_DATA[0].children[1]);
      await nextTick();

      const optionValues = wrapper.vm.parentOptions.map(o => o.value);

      expect(optionValues).not.toContain('child-2');
      expect(optionValues).toContain('root-1');
      expect(optionValues).toContain('child-1');
    });
  });

  describe('Delete Cascade Warning', () => {
    it('shows cascade warning with descendant count when node has children', async () => {
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.selectNode(TREE_DATA[0]);
      await nextTick();

      expect(wrapper.vm.selectedNodeChildCount).toBe(3);

      wrapper.vm.openDeleteDialog();
      await nextTick();

      expect(wrapper.vm.showDeleteDialog).toBe(true);
    });

    it('shows zero descendant count for leaf nodes', async () => {
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.selectNode(TREE_DATA[0].children[1]);
      await nextTick();

      expect(wrapper.vm.selectedNodeChildCount).toBe(0);
    });

    it('calls deleteNode API on confirm', async () => {
      orgApi.deleteNode.mockResolvedValue({});
      const wrapper = createWrapper();
      await flushPromises();

      wrapper.vm.selectNode(TREE_DATA[0].children[1]);
      await nextTick();

      await wrapper.vm.confirmDelete();
      await flushPromises();

      expect(orgApi.deleteNode).toHaveBeenCalledWith('child-2');
      expect(wrapper.vm.selectedNode).toBeNull();
      expect(wrapper.vm.showDeleteDialog).toBe(false);
    });
  });
});