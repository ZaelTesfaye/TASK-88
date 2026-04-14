<template>
  <div class="org-page">
    <div class="page-header">
      <h1 class="page-header__title">Organization Tree</h1>
      <div class="page-header__actions">
        <AppInput
          v-model="searchQuery"
          placeholder="Search nodes..."
          class="search-input"
        >
          <template #prefix>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <circle cx="7" cy="7" r="5" stroke="currentColor" stroke-width="1.5" />
              <path d="M11 11L14 14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
          </template>
        </AppInput>
        <AppButton @click="openCreateDialog(null)">Add Root Node</AppButton>
      </div>
    </div>

    <AppLoadingState v-if="loading" message="Loading organization tree..." />
    <AppErrorState v-else-if="error" :message="error" @retry="loadTree" />
    <AppEmptyState
      v-else-if="!treeData.length"
      title="No organization nodes"
      description="Create your first root node to start building the tree."
    >
      <template #action>
        <AppButton @click="openCreateDialog(null)">Add Root Node</AppButton>
      </template>
    </AppEmptyState>

    <div v-else class="org-content">
      <div class="tree-panel">
        <div class="tree-container">
          <OrgTreeNode
            v-for="node in filteredTree"
            :key="node.id"
            :node="node"
            :search="searchQuery"
            :selected-id="selectedNode?.id"
            :expanded-ids="expandedIds"
            @select="selectNode"
            @toggle="toggleExpand"
            @add-child="openCreateDialog"
          />
        </div>
      </div>

      <transition name="slide-right">
        <div v-if="selectedNode" class="detail-panel">
          <div class="detail-panel__header">
            <h3>Edit Node</h3>
            <button class="detail-panel__close" @click="selectedNode = null">
              <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
                <path d="M13.5 4.5L4.5 13.5M4.5 4.5L13.5 13.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
              </svg>
            </button>
          </div>
          <form class="detail-panel__form" @submit.prevent="saveNode">
            <AppInput v-model="editForm.name" label="Name" required :error="editErrors.name" />
            <div class="form-row">
              <AppInput v-model="editForm.level_code" label="Level Code" required />
              <AppInput v-model="editForm.level_label" label="Level Label" required />
            </div>
            <div class="form-row">
              <AppInput v-model="editForm.city" label="City" />
              <AppInput v-model="editForm.department" label="Department" />
            </div>
            <AppSelect
              v-model="editForm.parent_id"
              label="Parent Node"
              placeholder="None (root)"
              :options="parentOptions"
            />
            <div class="toggle-row">
              <label class="toggle-label">Active</label>
              <button
                type="button"
                class="toggle-switch"
                :class="{ active: editForm.is_active }"
                @click="editForm.is_active = !editForm.is_active"
              >
                <span class="toggle-switch__knob"></span>
              </button>
            </div>
            <div class="detail-panel__actions">
              <AppButton type="submit" :loading="saving">Save Changes</AppButton>
              <AppButton variant="danger" @click="openDeleteDialog">Delete</AppButton>
            </div>
          </form>
        </div>
      </transition>
    </div>

    <!-- Create Dialog -->
    <AppDialog v-model="showCreateDialog" :title="createParent ? `Add Child to ${createParent.name}` : 'Add Root Node'">
      <form @submit.prevent="createNode" class="dialog-form">
        <AppInput v-model="createForm.name" label="Name" required :error="createErrors.name" />
        <div class="form-row">
          <AppInput v-model="createForm.level_code" label="Level Code" required :error="createErrors.level_code" />
          <AppInput v-model="createForm.level_label" label="Level Label" required :error="createErrors.level_label" />
        </div>
        <div class="form-row">
          <AppInput v-model="createForm.city" label="City" />
          <AppInput v-model="createForm.department" label="Department" />
        </div>
      </form>
      <template #footer>
        <AppButton variant="secondary" @click="showCreateDialog = false">Cancel</AppButton>
        <AppButton :loading="creating" @click="createNode">Create Node</AppButton>
      </template>
    </AppDialog>

    <!-- Delete Confirmation Dialog -->
    <AppDialog v-model="showDeleteDialog" title="Delete Node" danger persistent>
      <div class="delete-warning">
        <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
          <circle cx="20" cy="20" r="16" stroke="currentColor" stroke-width="1.5" />
          <path d="M20 12V22" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          <circle cx="20" cy="28" r="1.5" fill="currentColor" />
        </svg>
        <p>Are you sure you want to delete <strong>{{ selectedNode?.name }}</strong>?</p>
        <p v-if="selectedNodeChildCount > 0" class="cascade-warning">
          This node has <strong>{{ selectedNodeChildCount }}</strong> descendant node(s).
          All child nodes will be permanently deleted as well.
        </p>
        <p class="delete-hint">This action cannot be undone.</p>
      </div>
      <template #footer>
        <AppButton variant="secondary" @click="showDeleteDialog = false">Cancel</AppButton>
        <AppButton variant="danger" :loading="deleting" @click="confirmDelete">Delete Node</AppButton>
      </template>
    </AppDialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, h, defineComponent } from 'vue';
import * as orgApi from '@/api/org.js';
import AppInput from '@/components/common/AppInput.vue';
import AppSelect from '@/components/common/AppSelect.vue';
import AppButton from '@/components/common/AppButton.vue';
import AppDialog from '@/components/common/AppDialog.vue';
import AppChip from '@/components/common/AppChip.vue';
import AppLoadingState from '@/components/common/AppLoadingState.vue';
import AppEmptyState from '@/components/common/AppEmptyState.vue';
import AppErrorState from '@/components/common/AppErrorState.vue';

// -- Recursive tree node sub-component --
const OrgTreeNode = defineComponent({
  name: 'OrgTreeNode',
  props: {
    node: { type: Object, required: true },
    depth: { type: Number, default: 0 },
    search: { type: String, default: '' },
    selectedId: { type: [String, Number], default: null },
    expandedIds: { type: Set, default: () => new Set() },
  },
  emits: ['select', 'toggle', 'add-child'],
  setup(props, { emit }) {
    const isExpanded = computed(() => props.expandedIds.has(props.node.id));
    const childCount = computed(() => countDescendants(props.node));
    const isSelected = computed(() => props.selectedId === props.node.id);

    return () => h('div', { class: 'tree-node', style: { '--depth': props.depth } }, [
      h('div', {
        class: ['tree-node__row', { selected: isSelected.value }],
        onClick: () => emit('select', props.node),
      }, [
        h('button', {
          class: ['tree-node__toggle', { invisible: !props.node.children?.length }],
          onClick: (e) => { e.stopPropagation(); emit('toggle', props.node.id); },
        }, [
          h('svg', {
            width: 14, height: 14, viewBox: '0 0 14 14', fill: 'none',
            class: { rotated: isExpanded.value },
            innerHTML: '<path d="M5 3L9 7L5 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>',
          }),
        ]),
        h('div', { class: 'tree-node__info' }, [
          h('span', { class: 'tree-node__name' }, props.node.name),
          h('span', { class: 'tree-node__meta' }, [
            props.node.level_label || props.node.level_code || '',
            props.node.city ? ` \u2022 ${props.node.city}` : '',
            props.node.department ? ` \u2022 ${props.node.department}` : '',
          ].join('')),
        ]),
        h(AppChip, {
          status: props.node.is_active ? 'active' : 'inactive',
          label: props.node.is_active ? 'Active' : 'Inactive',
          size: 'sm',
        }),
        childCount.value > 0
          ? h('span', { class: 'tree-node__count' }, `${childCount.value}`)
          : null,
        h('button', {
          class: 'tree-node__add-btn',
          title: 'Add child node',
          onClick: (e) => { e.stopPropagation(); emit('add-child', props.node); },
          innerHTML: '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 2V12M2 7H12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
        }),
      ]),
      isExpanded.value && props.node.children?.length
        ? h('div', { class: 'tree-node__children' },
            props.node.children.map((child) =>
              h(OrgTreeNode, {
                key: child.id,
                node: child,
                depth: props.depth + 1,
                search: props.search,
                selectedId: props.selectedId,
                expandedIds: props.expandedIds,
                onSelect: (n) => emit('select', n),
                onToggle: (id) => emit('toggle', id),
                onAddChild: (n) => emit('add-child', n),
              })
            )
          )
        : null,
    ]);
  },
});

// -- State --
const treeData = ref([]);
const loading = ref(false);
const error = ref('');
const searchQuery = ref('');
const selectedNode = ref(null);
const expandedIds = ref(new Set());
const saving = ref(false);
const creating = ref(false);
const deleting = ref(false);

// Edit form
const editForm = reactive({
  name: '', level_code: '', level_label: '', city: '', department: '', parent_id: '', is_active: true,
});
const editErrors = reactive({ name: '' });

// Create dialog
const showCreateDialog = ref(false);
const createParent = ref(null);
const createForm = reactive({ name: '', level_code: '', level_label: '', city: '', department: '' });
const createErrors = reactive({ name: '', level_code: '', level_label: '' });

// Delete dialog
const showDeleteDialog = ref(false);

// -- Computed --
const filteredTree = computed(() => {
  if (!searchQuery.value) return treeData.value;
  const idsToExpand = new Set();
  const result = filterTree(treeData.value, searchQuery.value.toLowerCase(), idsToExpand);
  if (idsToExpand.size > 0) {
    Promise.resolve().then(() => {
      const ids = new Set(expandedIds.value);
      for (const id of idsToExpand) ids.add(id);
      expandedIds.value = ids;
    });
  }
  return result;
});

const allFlatNodes = computed(() => flattenTree(treeData.value));

const parentOptions = computed(() => {
  if (!selectedNode.value) return [];
  const descIds = new Set();
  collectIds(selectedNode.value, descIds);
  return [
    { value: '', label: 'None (root node)' },
    ...allFlatNodes.value
      .filter((n) => !descIds.has(n.id))
      .map((n) => ({ value: n.id, label: `${'  '.repeat(n._depth || 0)}${n.name}` })),
  ];
});

const selectedNodeChildCount = computed(() => {
  if (!selectedNode.value) return 0;
  return countDescendants(selectedNode.value);
});

// -- Methods --
async function loadTree() {
  loading.value = true;
  error.value = '';
  try {
    const { data } = await orgApi.getOrgTree();
    treeData.value = Array.isArray(data) ? data : [data];
  } catch (e) {
    error.value = e?.response?.data?.message || 'Failed to load organization tree.';
  } finally {
    loading.value = false;
  }
}

function selectNode(node) {
  selectedNode.value = node;
  Object.assign(editForm, {
    name: node.name || '',
    level_code: node.level_code || '',
    level_label: node.level_label || '',
    city: node.city || '',
    department: node.department || '',
    parent_id: node.parent_id || '',
    is_active: node.is_active !== false,
  });
  editErrors.name = '';
}

function toggleExpand(nodeId) {
  const ids = new Set(expandedIds.value);
  if (ids.has(nodeId)) {
    ids.delete(nodeId);
  } else {
    ids.add(nodeId);
  }
  expandedIds.value = ids;
}

async function saveNode() {
  editErrors.name = '';
  if (!editForm.name.trim()) {
    editErrors.name = 'Name is required';
    return;
  }

  // Cycle prevention: cannot set parent to self or descendant
  if (editForm.parent_id && selectedNode.value) {
    const descIds = new Set();
    collectIds(selectedNode.value, descIds);
    if (descIds.has(editForm.parent_id)) {
      editErrors.name = 'Cannot set parent to self or a descendant node.';
      return;
    }
  }

  saving.value = true;
  try {
    await orgApi.updateNode(selectedNode.value.id, {
      name: editForm.name,
      level_code: editForm.level_code,
      level_label: editForm.level_label,
      city: editForm.city,
      department: editForm.department,
      parent_id: editForm.parent_id || null,
      is_active: editForm.is_active,
    });
    await loadTree();
  } catch (e) {
    editErrors.name = e?.response?.data?.message || 'Failed to save.';
  } finally {
    saving.value = false;
  }
}

function openCreateDialog(parentNode) {
  createParent.value = parentNode;
  Object.assign(createForm, { name: '', level_code: '', level_label: '', city: '', department: '' });
  Object.assign(createErrors, { name: '', level_code: '', level_label: '' });
  showCreateDialog.value = true;
}

async function createNode() {
  let valid = true;
  createErrors.name = '';
  createErrors.level_code = '';
  createErrors.level_label = '';

  if (!createForm.name.trim()) { createErrors.name = 'Required'; valid = false; }
  if (!createForm.level_code.trim()) { createErrors.level_code = 'Required'; valid = false; }
  if (!createForm.level_label.trim()) { createErrors.level_label = 'Required'; valid = false; }
  if (!valid) return;

  creating.value = true;
  try {
    await orgApi.createNode(createParent.value?.id || null, {
      name: createForm.name,
      level_code: createForm.level_code,
      level_label: createForm.level_label,
      city: createForm.city,
      department: createForm.department,
    });
    showCreateDialog.value = false;
    await loadTree();
    if (createParent.value) {
      const ids = new Set(expandedIds.value);
      ids.add(createParent.value.id);
      expandedIds.value = ids;
    }
  } catch (e) {
    createErrors.name = e?.response?.data?.message || 'Failed to create node.';
  } finally {
    creating.value = false;
  }
}

function openDeleteDialog() {
  showDeleteDialog.value = true;
}

async function confirmDelete() {
  deleting.value = true;
  try {
    await orgApi.deleteNode(selectedNode.value.id);
    showDeleteDialog.value = false;
    selectedNode.value = null;
    await loadTree();
  } catch (e) {
    // eslint-disable-next-line no-console
    console.error('Delete failed', e);
  } finally {
    deleting.value = false;
  }
}

// -- Helpers --
function filterTree(nodes, query, idsToExpand) {
  const result = [];
  for (const node of nodes) {
    const nameMatch = node.name?.toLowerCase().includes(query);
    const filteredChildren = node.children ? filterTree(node.children, query, idsToExpand) : [];
    if (nameMatch || filteredChildren.length > 0) {
      result.push({ ...node, children: filteredChildren.length > 0 ? filteredChildren : node.children });
      if (filteredChildren.length > 0) {
        idsToExpand.add(node.id);
      }
    }
  }
  return result;
}

function flattenTree(nodes, depth = 0) {
  const flat = [];
  for (const node of nodes) {
    flat.push({ ...node, _depth: depth });
    if (node.children) {
      flat.push(...flattenTree(node.children, depth + 1));
    }
  }
  return flat;
}

function collectIds(node, ids) {
  ids.add(node.id);
  if (node.children) {
    for (const child of node.children) {
      collectIds(child, ids);
    }
  }
}

function countDescendants(node) {
  let count = 0;
  if (node.children) {
    for (const child of node.children) {
      count += 1 + countDescendants(child);
    }
  }
  return count;
}

onMounted(loadTree);
</script>

<style lang="scss" scoped>
.org-page {
  max-width: $page-max-width;
  margin: 0 auto;
}

.search-input {
  width: 240px;
}

.org-content {
  display: flex;
  gap: $space-6;
  align-items: flex-start;
}

.tree-panel {
  flex: 1;
  min-width: 0;
  background: $color-neutral-0;
  border: 1px solid $border-color;
  border-radius: $border-radius-md;
  padding: $space-4;
  box-shadow: $shadow-xs;
}

.tree-container {
  min-height: 200px;
}

// Tree node styles (applied via non-scoped or deep since sub-component renders in same scope)
:deep(.tree-node) {
  padding-left: calc(var(--depth, 0) * 20px);
}

:deep(.tree-node__row) {
  display: flex;
  align-items: center;
  gap: $space-2;
  padding: $space-2 $space-3;
  border-radius: $border-radius-base;
  cursor: pointer;
  transition: background $transition-fast;

  &:hover {
    background: $color-neutral-50;
  }

  &.selected {
    background: $color-primary-50;
    border: 1px solid $color-primary-200;
  }
}

:deep(.tree-node__toggle) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: $border-radius-sm;
  color: $color-neutral-400;
  flex-shrink: 0;
  transition: all $transition-fast;

  &:hover {
    background: $color-neutral-100;
    color: $color-neutral-600;
  }

  &.invisible {
    visibility: hidden;
  }

  svg {
    transition: transform $transition-fast;

    &.rotated {
      transform: rotate(90deg);
    }
  }
}

:deep(.tree-node__info) {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

:deep(.tree-node__name) {
  font-size: $font-size-base;
  font-weight: $font-weight-medium;
  color: $color-neutral-800;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:deep(.tree-node__meta) {
  font-size: $font-size-xs;
  color: $color-neutral-400;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:deep(.tree-node__count) {
  font-size: $font-size-xs;
  color: $color-neutral-400;
  background: $color-neutral-100;
  padding: 1px 6px;
  border-radius: $border-radius-full;
  flex-shrink: 0;
}

:deep(.tree-node__add-btn) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: $border-radius-sm;
  color: $color-neutral-300;
  opacity: 0;
  transition: all $transition-fast;
  flex-shrink: 0;

  .tree-node__row:hover & {
    opacity: 1;
  }

  &:hover {
    background: $color-primary-50;
    color: $color-primary-500;
  }
}

:deep(.tree-node__children) {
  border-left: 1px solid $color-neutral-100;
  margin-left: 11px;
}

// Detail panel
.detail-panel {
  width: 360px;
  flex-shrink: 0;
  background: $color-neutral-0;
  border: 1px solid $border-color;
  border-radius: $border-radius-md;
  box-shadow: $shadow-xs;
  overflow: hidden;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: $space-4 $space-5;
    border-bottom: 1px solid $border-color;

    h3 {
      font-size: $font-size-md;
      font-weight: $font-weight-semibold;
      color: $color-neutral-800;
      margin: 0;
    }
  }

  &__close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border-radius: $border-radius-base;
    color: $color-neutral-400;
    transition: all $transition-fast;

    &:hover {
      background: $color-neutral-50;
      color: $color-neutral-600;
    }
  }

  &__form {
    padding: $space-5;
    display: flex;
    flex-direction: column;
    gap: $space-4;
  }

  &__actions {
    display: flex;
    gap: $space-3;
    padding-top: $space-3;
    border-top: 1px solid $border-color;
    margin-top: $space-2;
  }
}

.toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.toggle-label {
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  color: $color-neutral-700;
}

.toggle-switch {
  position: relative;
  width: 40px;
  height: 22px;
  background: $color-neutral-200;
  border-radius: $border-radius-full;
  transition: background $transition-fast;
  cursor: pointer;

  &.active {
    background: $color-success-500;
  }

  &__knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 18px;
    height: 18px;
    background: #fff;
    border-radius: $border-radius-full;
    box-shadow: $shadow-xs;
    transition: transform $transition-fast;

    .active & {
      transform: translateX(18px);
    }
  }
}

.dialog-form {
  display: flex;
  flex-direction: column;
  gap: $space-4;
}

.form-row {
  display: flex;
  gap: $space-4;

  > * {
    flex: 1;
  }
}

.delete-warning {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: $space-3;
  color: $color-danger-400;

  p {
    color: $color-neutral-700;
    font-size: $font-size-base;
    margin: 0;
  }

  strong {
    font-weight: $font-weight-semibold;
  }
}

.cascade-warning {
  background: $color-warning-50;
  border: 1px solid rgba($color-warning-500, 0.2);
  color: $color-warning-700 !important;
  padding: $space-3 $space-4;
  border-radius: $border-radius-base;
  font-size: $font-size-sm !important;
}

.delete-hint {
  font-size: $font-size-sm !important;
  color: $color-neutral-400 !important;
}

// Transition for detail panel
.slide-right-enter-active,
.slide-right-leave-active {
  transition: opacity $transition-base, transform $transition-base;
}

.slide-right-enter-from,
.slide-right-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

@media (max-width: 1024px) {
  .org-content {
    flex-direction: column;
  }

  .detail-panel {
    width: 100%;
  }

  .search-input {
    width: 180px;
  }
}
</style>
