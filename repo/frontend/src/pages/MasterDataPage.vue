<template>
  <div class="master-page">
    <div class="page-header">
      <h1 class="page-header__title">Master Data</h1>
      <div class="page-header__actions">
        <AppButton variant="outline" @click="showImportDialog = true">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M8 2V10M4 6L8 10L12 6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            <path d="M2 12V13C2 13.55 2.45 14 3 14H13C13.55 14 14 13.55 14 13V12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
          </svg>
          Import
        </AppButton>
        <AppButton @click="openRecordDialog(null)">New Record</AppButton>
      </div>
    </div>

    <!-- Entity Tabs -->
    <div class="entity-tabs">
      <router-link
        v-for="et in entityTypes"
        :key="et.key"
        :to="`/master/${et.key}`"
        class="entity-tab"
        :class="{ active: activeEntity === et.key }"
      >
        {{ et.label }}
      </router-link>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="toolbar__search">
        <AppInput v-model="searchInput" :placeholder="`Search ${activeEntityLabel}...`">
          <template #prefix>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <circle cx="7" cy="7" r="5" stroke="currentColor" stroke-width="1.5" />
              <path d="M11 11L14 14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
          </template>
        </AppInput>
      </div>
      <div class="toolbar__spacer"></div>
      <div v-if="selectedRows.length" class="bulk-actions">
        <span class="bulk-actions__count">{{ selectedRows.length }} selected</span>
        <AppButton size="sm" variant="outline" @click="bulkDeactivate">Deactivate</AppButton>
        <AppButton size="sm" variant="ghost" @click="selectedRows = []">Clear</AppButton>
      </div>
      <div v-if="versionInfo" class="version-badge">
        <span class="version-badge__label">Version:</span>
        <AppChip :status="versionInfo.status" :label="versionInfo.status" size="sm" />
      </div>
    </div>

    <!-- Duplicate Warning -->
    <div v-if="duplicateWarning" class="duplicate-banner">
      <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
        <path d="M9 1L17 15H1L9 1Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
        <path d="M9 6.5V10M9 12.5V12.51" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
      </svg>
      <span>{{ duplicateWarning }}</span>
      <button @click="duplicateWarning = ''" class="duplicate-banner__close">
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <path d="M10.5 3.5L3.5 10.5M3.5 3.5L10.5 10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        </svg>
      </button>
    </div>

    <!-- Data Table -->
    <AppLoadingState v-if="loading" variant="skeleton" :lines="8" />
    <AppErrorState v-else-if="loadError" :message="loadError" @retry="fetchRecords" />
    <AppEmptyState
      v-else-if="!rows.length && !searchDebounced"
      :title="`No ${activeEntityLabel} records`"
      :description="`Create your first ${activeEntityLabel} record to get started.`"
    >
      <template #action>
        <AppButton @click="openRecordDialog(null)">New {{ activeEntityLabel }}</AppButton>
      </template>
    </AppEmptyState>
    <AppEmptyState
      v-else-if="!rows.length && searchDebounced"
      title="No matching records"
      :description="`No ${activeEntityLabel} records match &quot;${searchDebounced}&quot;.`"
    />
    <template v-else>
      <AppTable
        :columns="activeColumns"
        :rows="rows"
        :loading="loading"
        :sort-key="sortKey"
        :sort-order="sortOrder"
        :current-page="currentPage"
        :total-pages="totalPages"
        :total-items="totalItems"
        row-clickable
        @sort="onSort"
        @page-change="onPageChange"
        @row-click="openRecordDialog"
      >
        <template #cell-status="{ value }">
          <AppChip :status="value" :label="value" size="sm" />
        </template>
        <template #cell-is_active="{ value }">
          <AppChip :status="value ? 'active' : 'inactive'" :label="value ? 'Active' : 'Inactive'" size="sm" />
        </template>
        <template #cell-hex_value="{ value }">
          <div v-if="value" class="color-cell">
            <span class="color-swatch" :style="{ background: value }"></span>
            <span>{{ value }}</span>
          </div>
          <span v-else>---</span>
        </template>
        <template #cell-select="{ row }">
          <input
            type="checkbox"
            :checked="selectedRows.includes(row.id)"
            @click.stop="toggleRowSelect(row)"
          />
        </template>
      </AppTable>
    </template>

    <!-- Record Create/Edit Dialog -->
    <AppDialog v-model="showRecordDialog" :title="editingRecord ? `Edit ${activeEntityLabel}` : `New ${activeEntityLabel}`" size="md">
      <form @submit.prevent="saveRecord" class="record-form">
        <template v-for="field in activeFields" :key="field.key">
          <AppSelect
            v-if="field.type === 'select'"
            v-model="recordForm[field.key]"
            :label="field.label"
            :required="field.required"
            :options="field.options"
            :placeholder="`Select ${field.label}`"
            :error="recordErrors[field.key]"
          />
          <AppInput
            v-else
            v-model="recordForm[field.key]"
            :label="field.label"
            :required="field.required"
            :placeholder="field.placeholder || `Enter ${field.label.toLowerCase()}`"
            :hint="field.hint"
            :error="recordErrors[field.key]"
            :type="field.type || 'text'"
          />
        </template>
      </form>
      <template #footer>
        <AppButton variant="secondary" @click="showRecordDialog = false">Cancel</AppButton>
        <AppButton :loading="saving" @click="saveRecord">{{ editingRecord ? 'Update' : 'Create' }}</AppButton>
      </template>
    </AppDialog>

    <!-- Deactivate Dialog -->
    <AppDialog v-model="showDeactivateDialog" title="Deactivate Record" persistent>
      <div class="deactivate-content">
        <div class="deactivate-warning">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <circle cx="10" cy="10" r="8" stroke="currentColor" stroke-width="1.5" />
            <path d="M10 6V11M10 13.5V13.51" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
          </svg>
          <p>Deactivating this record will hide it from all selection lists and active views. It can be reactivated later.</p>
        </div>
        <AppInput
          v-model="deactivateReason"
          label="Reason for deactivation"
          type="textarea"
          required
          placeholder="Explain why this record is being deactivated..."
          :error="deactivateReasonError"
          :rows="3"
        />
      </div>
      <template #footer>
        <AppButton variant="secondary" @click="showDeactivateDialog = false">Cancel</AppButton>
        <AppButton variant="danger" :loading="deactivating" @click="confirmDeactivate">Deactivate</AppButton>
      </template>
    </AppDialog>

    <!-- Import Dialog -->
    <AppDialog v-model="showImportDialog" title="Import Records" size="md" persistent>
      <div class="import-content">
        <p class="import-description">Upload a CSV or XLSX file to import {{ activeEntityLabel }} records. Maximum file size: 50 MB. Files must be UTF-8 encoded.</p>
        <AppFileUpload
          accept=".csv,.xlsx"
          :max-size="50 * 1024 * 1024"
          hint="CSV or XLSX, max 50 MB, UTF-8 encoded"
          :progress="importProgress"
          @file-selected="onImportFileSelected"
          @file-removed="importFile = null"
        />
        <div v-if="importResult" class="import-result">
          <div class="import-result__stat import-result__stat--success">
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <path d="M15 4.5L6.75 12.75L3 9" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
            <span>{{ importResult.successCount }} records imported</span>
          </div>
          <div v-if="importResult.errorCount > 0" class="import-result__stat import-result__stat--error">
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="1.5" />
              <path d="M9 5.5V9.5M9 12V12.01" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
            <span>{{ importResult.errorCount }} errors</span>
            <button v-if="importResult.errorReportUrl" class="import-result__download" @click="downloadErrorReport">
              Download Error Report
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <AppButton variant="secondary" @click="closeImportDialog">{{ importResult ? 'Done' : 'Cancel' }}</AppButton>
        <AppButton
          v-if="!importResult"
          :loading="importing"
          :disabled="!importFile"
          @click="executeImport"
        >Upload &amp; Import</AppButton>
      </template>
    </AppDialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import * as masterApi from '@/api/master.js';
import { useContextStore } from '@/stores/context.js';
import AppInput from '@/components/common/AppInput.vue';
import AppSelect from '@/components/common/AppSelect.vue';
import AppButton from '@/components/common/AppButton.vue';
import AppDialog from '@/components/common/AppDialog.vue';
import AppChip from '@/components/common/AppChip.vue';
import AppTable from '@/components/common/AppTable.vue';
import AppFileUpload from '@/components/common/AppFileUpload.vue';
import AppLoadingState from '@/components/common/AppLoadingState.vue';
import AppEmptyState from '@/components/common/AppEmptyState.vue';
import AppErrorState from '@/components/common/AppErrorState.vue';

const route = useRoute();
const router = useRouter();
const contextStore = useContextStore();

// ============================================================
// Entity Configuration
// ============================================================
const entityTypes = [
  { key: 'sku', label: 'SKU' },
  { key: 'color', label: 'Color' },
  { key: 'size', label: 'Size' },
  { key: 'season', label: 'Season' },
  { key: 'brand', label: 'Brand' },
  { key: 'supplier', label: 'Supplier' },
  { key: 'customer', label: 'Customer' },
];

const ENTITY_COLUMNS = {
  sku: [
    { key: 'select', label: '', width: '40px' },
    { key: 'code', label: 'Code', sortable: true },
    { key: 'description', label: 'Description', sortable: true },
    { key: 'category', label: 'Category', sortable: true },
    { key: 'status', label: 'Status', sortable: true, width: '120px' },
  ],
  color: [
    { key: 'select', label: '', width: '40px' },
    { key: 'code', label: 'Code', sortable: true },
    { key: 'name', label: 'Name', sortable: true },
    { key: 'hex_value', label: 'Color', sortable: false, width: '140px' },
    { key: 'status', label: 'Status', sortable: true, width: '120px' },
  ],
  size: [
    { key: 'select', label: '', width: '40px' },
    { key: 'code', label: 'Code', sortable: true },
    { key: 'label', label: 'Label', sortable: true },
    { key: 'sort_order', label: 'Sort Order', sortable: true, width: '120px' },
    { key: 'status', label: 'Status', sortable: true, width: '120px' },
  ],
  season: [
    { key: 'select', label: '', width: '40px' },
    { key: 'code', label: 'Code', sortable: true },
    { key: 'name', label: 'Name', sortable: true },
    { key: 'year', label: 'Year', sortable: true, width: '100px' },
    { key: 'status', label: 'Status', sortable: true, width: '120px' },
  ],
  brand: [
    { key: 'select', label: '', width: '40px' },
    { key: 'code', label: 'Code', sortable: true },
    { key: 'name', label: 'Name', sortable: true },
    { key: 'country', label: 'Country', sortable: true },
    { key: 'status', label: 'Status', sortable: true, width: '120px' },
  ],
  supplier: [
    { key: 'select', label: '', width: '40px' },
    { key: 'code', label: 'Code', sortable: true },
    { key: 'name', label: 'Name', sortable: true },
    { key: 'phone', label: 'Phone', sortable: false },
    { key: 'email', label: 'Email', sortable: true },
    { key: 'city', label: 'City', sortable: true },
    { key: 'status', label: 'Status', sortable: true, width: '120px' },
  ],
  customer: [
    { key: 'select', label: '', width: '40px' },
    { key: 'code', label: 'Code', sortable: true },
    { key: 'name', label: 'Name', sortable: true },
    { key: 'contact', label: 'Contact', sortable: true },
    { key: 'city', label: 'City', sortable: true },
    { key: 'status', label: 'Status', sortable: true, width: '120px' },
  ],
};

const ENTITY_FIELDS = {
  sku: [
    { key: 'code', label: 'Code', required: true, placeholder: 'e.g. SKU001ABCDEF', hint: '6-20 uppercase letters and digits', pattern: /^[A-Z0-9]{6,20}$/, patternMsg: 'Must be 6-20 uppercase letters/digits (A-Z, 0-9)' },
    { key: 'description', label: 'Description', required: true },
    { key: 'category', label: 'Category', required: true },
  ],
  color: [
    { key: 'code', label: 'Code', required: true },
    { key: 'name', label: 'Name', required: true },
    { key: 'hex_value', label: 'Hex Color', placeholder: '#FF5733', hint: 'e.g. #FF5733', pattern: /^#[0-9A-Fa-f]{6}$/, patternMsg: 'Must be a valid hex color (e.g. #FF5733)' },
  ],
  size: [
    { key: 'code', label: 'Code', required: true },
    { key: 'label', label: 'Label', required: true },
    { key: 'sort_order', label: 'Sort Order', type: 'number', required: true },
  ],
  season: [
    { key: 'code', label: 'Code', required: true, placeholder: 'e.g. SS2025 or FW2025', hint: 'Format: SS or FW followed by 4-digit year', pattern: /^(SS|FW)[0-9]{4}$/, patternMsg: 'Must match format SS#### or FW#### (e.g. SS2025)' },
    { key: 'name', label: 'Name', required: true },
    { key: 'year', label: 'Year', type: 'number', required: true },
  ],
  brand: [
    { key: 'code', label: 'Code', required: true },
    { key: 'name', label: 'Name', required: true },
    { key: 'country', label: 'Country', required: false },
  ],
  supplier: [
    { key: 'code', label: 'Code', required: true },
    { key: 'name', label: 'Name', required: true },
    { key: 'phone', label: 'Phone', placeholder: '(555) 123-4567', hint: 'Format: (###) ###-####', pattern: /^\([0-9]{3}\) [0-9]{3}-[0-9]{4}$/, patternMsg: 'Must match (###) ###-#### format' },
    { key: 'email', label: 'Email', type: 'email', pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/, patternMsg: 'Must be a valid email address' },
    { key: 'city', label: 'City' },
  ],
  customer: [
    { key: 'code', label: 'Code', required: true },
    { key: 'name', label: 'Name', required: true },
    { key: 'contact', label: 'Contact Person' },
    { key: 'city', label: 'City' },
  ],
};

// ============================================================
// State
// ============================================================
const activeEntity = computed(() => route.params.entity || 'sku');
const activeEntityLabel = computed(() => entityTypes.find((e) => e.key === activeEntity.value)?.label || 'Record');
const activeColumns = computed(() => ENTITY_COLUMNS[activeEntity.value] || []);
const activeFields = computed(() => ENTITY_FIELDS[activeEntity.value] || []);

const rows = ref([]);
const loading = ref(false);
const loadError = ref('');
const searchInput = ref('');
const searchDebounced = ref('');
const sortKey = ref('code');
const sortOrder = ref('asc');
const currentPage = ref(1);
const totalPages = ref(1);
const totalItems = ref(0);
const selectedRows = ref([]);
const versionInfo = ref(null);
const duplicateWarning = ref('');

// Dialogs
const showRecordDialog = ref(false);
const editingRecord = ref(null);
const recordForm = reactive({});
const recordErrors = reactive({});
const saving = ref(false);

const showDeactivateDialog = ref(false);
const deactivateTarget = ref(null);
const deactivateReason = ref('');
const deactivateReasonError = ref('');
const deactivating = ref(false);

const showImportDialog = ref(false);
const importFile = ref(null);
const importProgress = ref(null);
const importing = ref(false);
const importResult = ref(null);

const PAGE_SIZE = 25;
let searchTimer = null;

// ============================================================
// Watchers
// ============================================================
watch(() => route.params.entity, () => {
  resetState();
  fetchRecords();
}, { immediate: false });

watch(searchInput, (val) => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    searchDebounced.value = val;
    currentPage.value = 1;
    fetchRecords();
  }, 300);
});

// ============================================================
// Data Fetching
// ============================================================
async function fetchRecords() {
  loading.value = true;
  loadError.value = '';
  duplicateWarning.value = '';
  try {
    const params = {
      page: currentPage.value,
      pageSize: PAGE_SIZE,
      sortKey: sortKey.value,
      sortOrder: sortOrder.value,
      ...(searchDebounced.value ? { search: searchDebounced.value } : {}),
      ...(contextStore.scopeFilter.nodeId ? { nodeId: contextStore.scopeFilter.nodeId } : {}),
    };
    const { data } = await masterApi.getMasterRecords(activeEntity.value, params);
    rows.value = data.records || data.items || [];
    totalItems.value = data.total || data.totalItems || rows.value.length;
    totalPages.value = data.totalPages || Math.ceil(totalItems.value / PAGE_SIZE) || 1;
    versionInfo.value = data.version || null;

    // Check for duplicates
    if (data.duplicates && data.duplicates.length > 0) {
      duplicateWarning.value = `${data.duplicates.length} potential duplicate(s) detected. Review records before proceeding.`;
    }
  } catch (e) {
    loadError.value = e?.response?.data?.message || 'Failed to load records. Please try again.';
  } finally {
    loading.value = false;
  }
}

function resetState() {
  rows.value = [];
  searchInput.value = '';
  searchDebounced.value = '';
  sortKey.value = 'code';
  sortOrder.value = 'asc';
  currentPage.value = 1;
  selectedRows.value = [];
  duplicateWarning.value = '';
  loadError.value = '';
}

function onSort({ key, order }) {
  sortKey.value = key;
  sortOrder.value = order;
  fetchRecords();
}

function onPageChange(page) {
  currentPage.value = page;
  fetchRecords();
}

function toggleRowSelect(row) {
  const idx = selectedRows.value.indexOf(row.id);
  if (idx >= 0) {
    selectedRows.value.splice(idx, 1);
  } else {
    selectedRows.value.push(row.id);
  }
}

// ============================================================
// Record Dialog
// ============================================================
function openRecordDialog(row) {
  editingRecord.value = row;
  const fields = activeFields.value;
  for (const f of fields) {
    recordForm[f.key] = row ? (row[f.key] ?? '') : '';
    recordErrors[f.key] = '';
  }
  showRecordDialog.value = true;
}

function validateRecord() {
  let valid = true;
  for (const f of activeFields.value) {
    recordErrors[f.key] = '';
    const val = String(recordForm[f.key] ?? '').trim();

    if (f.required && !val) {
      recordErrors[f.key] = `${f.label} is required`;
      valid = false;
    } else if (val && f.pattern && !f.pattern.test(val)) {
      recordErrors[f.key] = f.patternMsg || `Invalid format`;
      valid = false;
    }
  }
  return valid;
}

async function saveRecord() {
  if (!validateRecord()) return;

  saving.value = true;
  duplicateWarning.value = '';

  const payload = {};
  for (const f of activeFields.value) {
    payload[f.key] = recordForm[f.key];
  }

  try {
    if (editingRecord.value) {
      await masterApi.updateRecord(activeEntity.value, editingRecord.value.id, payload);
    } else {
      await masterApi.createRecord(activeEntity.value, payload);
    }
    showRecordDialog.value = false;
    await fetchRecords();
  } catch (e) {
    const data = e?.response?.data;
    if (data?.duplicates && data.duplicates.length > 0) {
      duplicateWarning.value = `Duplicate match found: ${data.duplicates.map((d) => d.code || d.name).join(', ')}. Record was still saved.`;
    }
    if (data?.field) {
      recordErrors[data.field] = data.message || 'Validation error';
    } else {
      recordErrors[activeFields.value[0]?.key] = data?.message || 'Failed to save record.';
    }
  } finally {
    saving.value = false;
  }
}

// ============================================================
// Deactivate
// ============================================================
function bulkDeactivate() {
  if (selectedRows.value.length === 1) {
    const row = rows.value.find((r) => r.id === selectedRows.value[0]);
    if (row) openDeactivateDialog(row);
  }
  // For multiple: deactivate first selected for simplicity; in production, iterate
  if (selectedRows.value.length > 1) {
    const row = rows.value.find((r) => r.id === selectedRows.value[0]);
    if (row) openDeactivateDialog(row);
  }
}

function openDeactivateDialog(row) {
  deactivateTarget.value = row || editingRecord.value;
  deactivateReason.value = '';
  deactivateReasonError.value = '';
  showDeactivateDialog.value = true;
}

async function confirmDeactivate() {
  if (!deactivateReason.value.trim()) {
    deactivateReasonError.value = 'A reason is required to deactivate a record.';
    return;
  }
  deactivating.value = true;
  try {
    await masterApi.deactivateRecord(activeEntity.value, deactivateTarget.value.id, deactivateReason.value);
    showDeactivateDialog.value = false;
    selectedRows.value = [];
    await fetchRecords();
  } catch (e) {
    deactivateReasonError.value = e?.response?.data?.message || 'Failed to deactivate record.';
  } finally {
    deactivating.value = false;
  }
}

// ============================================================
// Import
// ============================================================
function onImportFileSelected(file) {
  importFile.value = file;
  importResult.value = null;
  importProgress.value = null;
}

async function executeImport() {
  if (!importFile.value) return;
  importing.value = true;
  importProgress.value = 0;
  importResult.value = null;

  try {
    const { data } = await masterApi.importRecords(
      activeEntity.value,
      importFile.value,
      (pct) => { importProgress.value = pct; }
    );
    importResult.value = {
      successCount: data.successCount || data.imported || 0,
      errorCount: data.errorCount || data.errors || 0,
      errorReportUrl: data.errorReportUrl || null,
    };
    importProgress.value = 100;
    await fetchRecords();
  } catch (e) {
    importResult.value = {
      successCount: 0,
      errorCount: 1,
      errorReportUrl: null,
    };
  } finally {
    importing.value = false;
  }
}

function downloadErrorReport() {
  if (importResult.value?.errorReportUrl) {
    window.open(importResult.value.errorReportUrl, '_blank');
  }
}

function closeImportDialog() {
  showImportDialog.value = false;
  importFile.value = null;
  importProgress.value = null;
  importResult.value = null;
}

// ============================================================
// Init
// ============================================================
onMounted(() => {
  if (!route.params.entity) {
    router.replace('/master/sku');
  }
  fetchRecords();
});
</script>

<style lang="scss" scoped>
.master-page {
  max-width: $page-max-width;
  margin: 0 auto;
}

// Entity Tabs
.entity-tabs {
  display: flex;
  gap: 2px;
  padding: 3px;
  background: $color-neutral-100;
  border-radius: $border-radius-md;
  margin-bottom: $space-5;
  overflow-x: auto;
}

.entity-tab {
  padding: $space-2 $space-4;
  font-size: $font-size-base;
  font-weight: $font-weight-medium;
  color: $color-neutral-600;
  border-radius: $border-radius-base;
  white-space: nowrap;
  text-decoration: none;
  transition: all $transition-fast;

  &:hover {
    color: $color-neutral-800;
    background: rgba(255, 255, 255, 0.6);
    text-decoration: none;
  }

  &.active {
    color: $color-neutral-900;
    background: $color-neutral-0;
    box-shadow: $shadow-xs;
  }
}

// Toolbar
.toolbar {
  display: flex;
  align-items: center;
  gap: $space-3;
  margin-bottom: $space-4;
  flex-wrap: wrap;

  &__search {
    width: 300px;
    flex-shrink: 0;
  }

  &__spacer {
    flex: 1;
  }
}

.bulk-actions {
  display: flex;
  align-items: center;
  gap: $space-2;
  padding: $space-1 $space-3;
  background: $color-primary-50;
  border: 1px solid $color-primary-200;
  border-radius: $border-radius-base;

  &__count {
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    color: $color-primary-600;
  }
}

.version-badge {
  display: flex;
  align-items: center;
  gap: $space-2;

  &__label {
    font-size: $font-size-sm;
    color: $color-neutral-500;
  }
}

// Duplicate Banner
.duplicate-banner {
  display: flex;
  align-items: center;
  gap: $space-3;
  padding: $space-3 $space-4;
  background: $color-warning-50;
  border: 1px solid rgba($color-warning-500, 0.25);
  border-radius: $border-radius-base;
  color: $color-warning-700;
  font-size: $font-size-base;
  margin-bottom: $space-4;

  svg {
    flex-shrink: 0;
  }

  span {
    flex: 1;
  }

  &__close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: $border-radius-sm;
    color: $color-warning-600;
    flex-shrink: 0;
    transition: all $transition-fast;

    &:hover {
      background: rgba($color-warning-500, 0.15);
    }
  }
}

// Color cell
.color-cell {
  display: flex;
  align-items: center;
  gap: $space-2;
}

.color-swatch {
  width: 18px;
  height: 18px;
  border-radius: $border-radius-sm;
  border: 1px solid $border-color;
  flex-shrink: 0;
}

// Record form
.record-form {
  display: flex;
  flex-direction: column;
  gap: $space-4;
}

// Deactivate
.deactivate-content {
  display: flex;
  flex-direction: column;
  gap: $space-4;
}

.deactivate-warning {
  display: flex;
  gap: $space-3;
  padding: $space-3 $space-4;
  background: $color-warning-50;
  border-radius: $border-radius-base;
  color: $color-warning-700;
  font-size: $font-size-base;

  svg {
    flex-shrink: 0;
    margin-top: 1px;
  }

  p {
    margin: 0;
  }
}

// Import
.import-content {
  display: flex;
  flex-direction: column;
  gap: $space-4;
}

.import-description {
  font-size: $font-size-base;
  color: $color-neutral-600;
  margin: 0;
}

.import-result {
  display: flex;
  flex-direction: column;
  gap: $space-2;
  padding: $space-4;
  background: $color-neutral-25;
  border-radius: $border-radius-base;
  border: 1px solid $border-color;

  &__stat {
    display: flex;
    align-items: center;
    gap: $space-2;
    font-size: $font-size-base;
    font-weight: $font-weight-medium;

    &--success {
      color: $color-success-600;
    }

    &--error {
      color: $color-danger-600;
    }
  }

  &__download {
    margin-left: $space-2;
    font-size: $font-size-sm;
    color: $color-primary-500;
    font-weight: $font-weight-medium;
    text-decoration: underline;
    cursor: pointer;

    &:hover {
      color: $color-primary-600;
    }
  }
}

@media (max-width: 768px) {
  .entity-tabs {
    margin-left: -$space-4;
    margin-right: -$space-4;
    border-radius: 0;
    padding: 3px $space-4;
  }

  .toolbar {
    &__search {
      width: 100%;
    }
  }
}
</style>
