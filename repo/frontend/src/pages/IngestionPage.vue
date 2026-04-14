<template>
  <div class="page-content ingestion-page">
    <AppToast ref="toast" />

    <div class="page-header">
      <h2 class="page-header__title">Ingestion</h2>
      <div class="page-header__actions">
        <AppButton v-if="activeTab === 'sources'" variant="primary" @click="openAddSource">Add Source</AppButton>
        <AppButton v-if="activeTab === 'jobs'" variant="primary" @click="showRunJobDialog = true">Run Job</AppButton>
      </div>
    </div>

    <!-- Tabs -->
    <div class="tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >{{ tab.label }}</button>
    </div>

    <!-- Sources Tab -->
    <div v-if="activeTab === 'sources'" class="tab-panel">
      <AppLoadingState v-if="sourcesLoading" message="Loading sources..." />
      <AppErrorState v-else-if="sourcesError" :message="sourcesError" retryable @retry="loadSources" />
      <AppEmptyState v-else-if="sources.length === 0" title="No sources configured" description="Add an ingestion source to get started.">
        <template #action><AppButton variant="primary" @click="openAddSource">Add Source</AppButton></template>
      </AppEmptyState>

      <div v-else class="source-grid">
        <div v-for="src in sources" :key="src.id" class="source-card card">
          <div class="source-card__header">
            <div class="source-card__name">
              <span class="source-card__health" :class="`source-card__health--${src.healthStatus || 'unknown'}`" />
              <h4>{{ src.name }}</h4>
            </div>
            <AppChip :label="src.source_type || src.type" variant="info" size="sm" />
          </div>
          <div class="source-card__body">
            <div class="source-card__meta">
              <span class="text-sm text-muted">Last sync:</span>
              <span class="text-sm">{{ src.lastSyncAt ? new Date(src.lastSyncAt).toLocaleString() : 'Never' }}</span>
            </div>
            <div class="source-card__meta">
              <span class="text-sm text-muted">Status:</span>
              <AppChip :status="src.enabled ? 'active' : 'inactive'" :label="src.enabled ? 'Enabled' : 'Disabled'" size="sm" />
            </div>
          </div>
          <div class="source-card__actions">
            <AppButton variant="ghost" size="sm" @click="openEditSource(src)">Edit</AppButton>
            <AppButton variant="ghost" size="sm" @click="checkHealth(src)">Check Health</AppButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Job Queue Tab -->
    <div v-if="activeTab === 'jobs'" class="tab-panel">
      <AppLoadingState v-if="jobsLoading" message="Loading jobs..." />
      <AppErrorState v-else-if="jobsError" :message="jobsError" retryable @retry="loadJobs" />
      <AppEmptyState v-else-if="jobs.length === 0" title="No jobs in queue" description="Run a job to start ingesting data." />

      <template v-else>
        <AppTable
          :columns="jobColumns"
          :rows="jobs"
          :loading="jobsLoading"
          :current-page="jobPage"
          :total-pages="jobTotalPages"
          @page-change="p => { jobPage = p; loadJobs(); }"
        >
          <template #cell-state="{ row }">
            <AppChip
              :status="row.state"
              :label="row.state"
              :class="{ 'chip-pulse': row.state === 'running' }"
            />
          </template>
          <template #cell-progress="{ row }">
            <div class="progress-cell">
              <div class="progress-mini">
                <div class="progress-mini__fill" :style="{ width: jobProgress(row) + '%' }" />
              </div>
              <span class="text-xs text-muted">{{ row.recordsProcessed || 0 }} / {{ row.totalEstimate || '?' }}</span>
            </div>
          </template>
          <template #cell-dependencies="{ row }">
            <span v-if="row.blockedBy && row.blockedBy.length > 0" class="text-xs text-warning">
              Blocked by: {{ row.blockedBy.join(', ') }}
            </span>
            <span v-else class="text-xs text-muted">None</span>
          </template>
          <template #cell-retry="{ row }">
            <div v-if="row.retryCount > 0" class="retry-info">
              <span class="text-xs">Retries: {{ row.retryCount }}</span>
              <span v-if="row.nextRetryAt" class="text-xs text-muted">Next: {{ new Date(row.nextRetryAt).toLocaleString() }}</span>
            </div>
            <span v-else class="text-xs text-muted">--</span>
          </template>
          <template #cell-actions="{ row }">
            <AppButton
              v-if="row.state === 'failed'"
              variant="outline"
              size="sm"
              @click="openAckDialog(row)"
            >Acknowledge</AppButton>
          </template>
        </AppTable>
      </template>
    </div>

    <!-- Connectors Tab -->
    <div v-if="activeTab === 'connectors'" class="tab-panel">
      <AppLoadingState v-if="connectorsLoading" message="Loading connectors..." />
      <AppErrorState v-else-if="connectorsError" :message="connectorsError" retryable @retry="loadConnectors" />
      <AppEmptyState v-else-if="connectors.length === 0" title="No connectors" description="No ingestion connectors are configured." />

      <div v-else class="connector-grid">
        <div v-for="conn in connectors" :key="conn.type" class="connector-card card">
          <div class="connector-card__header">
            <h4>{{ conn.type }}</h4>
            <span class="connector-card__health" :class="`connector-card__health--${conn.healthStatus || 'unknown'}`">
              {{ conn.healthStatus || 'Unknown' }}
            </span>
          </div>
          <div class="connector-card__body">
            <div v-if="conn.capabilities && conn.capabilities.length" class="connector-card__caps">
              <AppChip v-for="cap in conn.capabilities" :key="cap" :label="cap" variant="neutral" size="sm" />
            </div>
            <p v-if="conn.lastHealthCheck" class="text-xs text-muted mt-2">Last check: {{ new Date(conn.lastHealthCheck).toLocaleString() }}</p>
          </div>
          <div class="connector-card__actions">
            <AppButton variant="outline" size="sm" :loading="conn._checking" @click="checkConnectorHealth(conn)">Check Health</AppButton>
            <AppButton v-if="conn.configSchema" variant="ghost" size="sm" @click="viewConnectorSchema(conn)">View Schema</AppButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Add/Edit Source Dialog -->
    <AppDialog v-model="showSourceDialog" :title="editingSource ? 'Edit Source' : 'Add Source'" size="lg" persistent>
      <div class="form-grid">
        <AppInput v-model="sourceForm.name" label="Name" required :error="sourceFormErrors.name" />
        <AppSelect v-model="sourceForm.source_type" label="Source Type" required :options="sourceTypes" placeholder="Select type..." :error="sourceFormErrors.source_type" />
        <AppInput v-model="sourceForm.config" label="Config (JSON)" type="textarea" :rows="5" hint="Paste connection configuration JSON" :error="sourceFormErrors.config" />
        <AppInput v-model="sourceForm.validationRules" label="Validation Rules (JSON)" type="textarea" :rows="3" />
        <AppInput v-model="sourceForm.mappingRules" label="Mapping Rules (JSON)" type="textarea" :rows="3" />
        <div class="form-toggle">
          <label class="toggle-label">
            <input type="checkbox" v-model="sourceForm.enabled" />
            <span>Enabled</span>
          </label>
        </div>
      </div>
      <template #footer>
        <AppButton variant="outline" @click="showSourceDialog = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="savingSource" @click="saveSource">{{ editingSource ? 'Update' : 'Create' }}</AppButton>
      </template>
    </AppDialog>

    <!-- Run Job Dialog -->
    <AppDialog v-model="showRunJobDialog" title="Run Ingestion Job" size="md">
      <AppSelect v-model="runJobForm.sourceId" label="Source" :options="sourceOptions" placeholder="Select source..." required />
      <AppSelect v-model="runJobForm.mode" label="Mode" :options="['incremental', 'backfill']" class="mt-4" />
      <template #footer>
        <AppButton variant="outline" @click="showRunJobDialog = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="runningJob" @click="runNewJob">Run</AppButton>
      </template>
    </AppDialog>

    <!-- Acknowledge Dialog -->
    <AppDialog v-model="showAckDialog" title="Acknowledge Failed Job" size="md" danger>
      <div v-if="ackJob" class="ack-details">
        <div class="ack-details__error">
          <h5>Error Details</h5>
          <pre class="error-pre">{{ ackJob.errorDetails || ackJob.error || 'No details available' }}</pre>
        </div>
        <AppInput v-model="ackReason" label="Acknowledgment Reason" type="textarea" :rows="3" required placeholder="Describe how you will address this failure..." :error="ackReasonError" />
      </div>
      <template #footer>
        <AppButton variant="outline" @click="showAckDialog = false">Cancel</AppButton>
        <AppButton variant="danger" :loading="acknowledging" @click="acknowledgeFailedJob">Acknowledge</AppButton>
      </template>
    </AppDialog>

    <!-- Connector Schema Viewer -->
    <AppDialog v-model="showSchemaDialog" title="Connector Config Schema" size="lg">
      <pre class="schema-pre">{{ connectorSchemaContent }}</pre>
    </AppDialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue';
import * as ingestionApi from '@/api/ingestion.js';
import AppButton from '@/components/common/AppButton.vue';
import AppInput from '@/components/common/AppInput.vue';
import AppSelect from '@/components/common/AppSelect.vue';
import AppChip from '@/components/common/AppChip.vue';
import AppTable from '@/components/common/AppTable.vue';
import AppDialog from '@/components/common/AppDialog.vue';
import AppLoadingState from '@/components/common/AppLoadingState.vue';
import AppErrorState from '@/components/common/AppErrorState.vue';
import AppEmptyState from '@/components/common/AppEmptyState.vue';
import AppToast from '@/components/common/AppToast.vue';

const toast = ref(null);
const tabs = [
  { key: 'sources', label: 'Sources' },
  { key: 'jobs', label: 'Job Queue' },
  { key: 'connectors', label: 'Connectors' },
];
const activeTab = ref('sources');
const sourceTypes = ['folder', 'share', 'db', 'api', 's3'];

// ---- Sources ----
const sources = ref([]);
const sourcesLoading = ref(false);
const sourcesError = ref('');

async function loadSources() {
  sourcesLoading.value = true;
  sourcesError.value = '';
  try {
    const { data } = await ingestionApi.getSources();
    sources.value = Array.isArray(data) ? data : data.items || [];
  } catch (err) {
    sourcesError.value = err.message || 'Failed to load sources';
  } finally {
    sourcesLoading.value = false;
  }
}

const sourceOptions = computed(() => sources.value.map(s => ({ value: s.id, label: s.name })));

// Source dialog
const showSourceDialog = ref(false);
const editingSource = ref(null);
const savingSource = ref(false);
const sourceForm = ref({ name: '', source_type: '', config: '', validationRules: '', mappingRules: '', enabled: true });
const sourceFormErrors = ref({});

function openAddSource() {
  editingSource.value = null;
  sourceForm.value = { name: '', source_type: '', config: '', validationRules: '', mappingRules: '', enabled: true };
  sourceFormErrors.value = {};
  showSourceDialog.value = true;
}

function openEditSource(src) {
  editingSource.value = src;
  sourceForm.value = {
    name: src.name,
    source_type: src.source_type || src.type || '',
    config: typeof src.config === 'object' ? JSON.stringify(src.config, null, 2) : src.config || '',
    validationRules: typeof src.validationRules === 'object' ? JSON.stringify(src.validationRules, null, 2) : src.validationRules || '',
    mappingRules: typeof src.mappingRules === 'object' ? JSON.stringify(src.mappingRules, null, 2) : src.mappingRules || '',
    enabled: src.enabled !== false,
  };
  sourceFormErrors.value = {};
  showSourceDialog.value = true;
}

function validateSourceForm() {
  const errors = {};
  if (!sourceForm.value.name.trim()) errors.name = 'Name is required';
  if (!sourceForm.value.source_type) errors.source_type = 'Source type is required';
  if (sourceForm.value.config) {
    try { JSON.parse(sourceForm.value.config); } catch { errors.config = 'Invalid JSON'; }
  }
  sourceFormErrors.value = errors;
  return Object.keys(errors).length === 0;
}

async function saveSource() {
  if (!validateSourceForm()) return;
  savingSource.value = true;
  const payload = {
    name: sourceForm.value.name,
    source_type: sourceForm.value.source_type,
    config: sourceForm.value.config ? JSON.parse(sourceForm.value.config) : {},
    validationRules: sourceForm.value.validationRules ? JSON.parse(sourceForm.value.validationRules) : null,
    mappingRules: sourceForm.value.mappingRules ? JSON.parse(sourceForm.value.mappingRules) : null,
    enabled: sourceForm.value.enabled,
  };
  try {
    if (editingSource.value) {
      await ingestionApi.updateSource(editingSource.value.id, payload);
      toast.value?.addToast({ message: 'Source updated', type: 'success' });
    } else {
      await ingestionApi.createSource(payload);
      toast.value?.addToast({ message: 'Source created', type: 'success' });
    }
    showSourceDialog.value = false;
    await loadSources();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Save failed', type: 'error' });
  } finally {
    savingSource.value = false;
  }
}

async function checkHealth(src) {
  try {
    const { data } = await ingestionApi.getConnectorHealth(src.id);
    src.healthStatus = data.status || data.healthStatus || 'healthy';
    toast.value?.addToast({ message: `${src.name}: ${src.healthStatus}`, type: src.healthStatus === 'healthy' ? 'success' : 'warning' });
  } catch (err) {
    src.healthStatus = 'error';
    toast.value?.addToast({ message: `Health check failed: ${err.message}`, type: 'error' });
  }
}

// ---- Jobs ----
const jobs = ref([]);
const jobsLoading = ref(false);
const jobsError = ref('');
const jobPage = ref(1);
const jobTotalPages = ref(1);
let jobRefreshTimer = null;

const jobColumns = [
  { key: 'id', label: 'ID', width: '80px' },
  { key: 'sourceName', label: 'Source' },
  { key: 'mode', label: 'Mode' },
  { key: 'priority', label: 'Priority', width: '80px' },
  { key: 'state', label: 'State' },
  { key: 'progress', label: 'Progress', width: '180px' },
  { key: 'dependencies', label: 'Dependencies' },
  { key: 'retry', label: 'Retry Info' },
  { key: 'created_at', label: 'Created' },
  { key: 'actions', label: '', width: '120px' },
];

async function loadJobs() {
  jobsLoading.value = true;
  jobsError.value = '';
  try {
    const { data } = await ingestionApi.getJobs({ page: jobPage.value });
    const list = Array.isArray(data) ? data : data.items || [];
    jobs.value = list;
    jobTotalPages.value = data.totalPages || 1;
    scheduleJobRefresh(list);
  } catch (err) {
    jobsError.value = err.message || 'Failed to load jobs';
  } finally {
    jobsLoading.value = false;
  }
}

function jobProgress(row) {
  if (!row.totalEstimate || row.totalEstimate === 0) return 0;
  return Math.min(100, Math.round((row.recordsProcessed / row.totalEstimate) * 100));
}

function scheduleJobRefresh(list) {
  clearInterval(jobRefreshTimer);
  if (list.some(j => j.state === 'running')) {
    jobRefreshTimer = setInterval(loadJobs, 10000);
  }
}

// Run Job
const showRunJobDialog = ref(false);
const runJobForm = ref({ sourceId: '', mode: 'incremental' });
const runningJob = ref(false);

async function runNewJob() {
  if (!runJobForm.value.sourceId) return;
  runningJob.value = true;
  try {
    await ingestionApi.runJob(runJobForm.value.sourceId);
    toast.value?.addToast({ message: 'Job started', type: 'success' });
    showRunJobDialog.value = false;
    activeTab.value = 'jobs';
    await loadJobs();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Failed to start job', type: 'error' });
  } finally {
    runningJob.value = false;
  }
}

// Acknowledge
const showAckDialog = ref(false);
const ackJob = ref(null);
const ackReason = ref('');
const ackReasonError = ref('');
const acknowledging = ref(false);

function openAckDialog(job) {
  ackJob.value = job;
  ackReason.value = '';
  ackReasonError.value = '';
  showAckDialog.value = true;
}

async function acknowledgeFailedJob() {
  if (!ackReason.value.trim()) { ackReasonError.value = 'Reason is required'; return; }
  acknowledging.value = true;
  try {
    await ingestionApi.acknowledgeJob(ackJob.value.id, ackReason.value);
    toast.value?.addToast({ message: 'Job acknowledged', type: 'success' });
    showAckDialog.value = false;
    await loadJobs();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Acknowledge failed', type: 'error' });
  } finally {
    acknowledging.value = false;
  }
}

// ---- Connectors ----
const connectors = ref([]);
const connectorsLoading = ref(false);
const connectorsError = ref('');
const showSchemaDialog = ref(false);
const connectorSchemaContent = ref('');

async function loadConnectors() {
  connectorsLoading.value = true;
  connectorsError.value = '';
  try {
    const { data } = await ingestionApi.getCapabilities();
    connectors.value = Array.isArray(data) ? data : data.items || data.connectors || [];
  } catch (err) {
    connectorsError.value = err.message || 'Failed to load connectors';
  } finally {
    connectorsLoading.value = false;
  }
}

async function checkConnectorHealth(conn) {
  conn._checking = true;
  try {
    const { data } = await ingestionApi.getConnectorHealth(conn.id || conn.type);
    conn.healthStatus = data.status || 'healthy';
    conn.lastHealthCheck = new Date().toISOString();
    toast.value?.addToast({ message: `${conn.type}: ${conn.healthStatus}`, type: conn.healthStatus === 'healthy' ? 'success' : 'warning' });
  } catch (err) {
    conn.healthStatus = 'error';
    toast.value?.addToast({ message: `Health check failed: ${err.message}`, type: 'error' });
  } finally {
    conn._checking = false;
  }
}

function viewConnectorSchema(conn) {
  connectorSchemaContent.value = typeof conn.configSchema === 'object'
    ? JSON.stringify(conn.configSchema, null, 2)
    : conn.configSchema || 'No schema available';
  showSchemaDialog.value = true;
}

// ---- Tab switching auto-load ----
watch(activeTab, (tab) => {
  if (tab === 'sources' && sources.value.length === 0) loadSources();
  if (tab === 'jobs') loadJobs();
  if (tab === 'connectors' && connectors.value.length === 0) loadConnectors();
});

onMounted(() => { loadSources(); });

onBeforeUnmount(() => { clearInterval(jobRefreshTimer); });
</script>

<style lang="scss" scoped>
.tabs {
  display: flex;
  gap: $space-1;
  border-bottom: 1px solid $border-color;
  margin-bottom: $space-6;
}

.tab-btn {
  padding: $space-3 $space-5;
  font-size: $font-size-base;
  font-weight: $font-weight-medium;
  color: $color-neutral-500;
  border-bottom: 2px solid transparent;
  transition: all $transition-fast;
  margin-bottom: -1px;

  &:hover { color: $color-neutral-700; }
  &--active {
    color: $color-primary-600;
    border-bottom-color: $color-primary-500;
  }
}

.tab-panel { min-height: 300px; }

// Sources
.source-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: $space-4;
}

.source-card {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: $space-4 $space-5;
    border-bottom: 1px solid $border-color;
  }
  &__name {
    display: flex;
    align-items: center;
    gap: $space-2;
    h4 { font-size: $font-size-md; margin: 0; }
  }
  &__health {
    width: 10px; height: 10px; border-radius: $border-radius-full; flex-shrink: 0;
    &--healthy { background: $color-success-500; }
    &--degraded { background: $color-warning-500; }
    &--error, &--unhealthy { background: $color-danger-500; }
    &--unknown { background: $color-neutral-300; }
  }
  &__body { padding: $space-4 $space-5; }
  &__meta { display: flex; justify-content: space-between; margin-bottom: $space-2; }
  &__actions {
    display: flex; gap: $space-2; padding: $space-3 $space-5; border-top: 1px solid $border-color;
  }
}

// Jobs
.progress-cell {
  display: flex;
  flex-direction: column;
  gap: $space-1;
}

.progress-mini {
  height: 6px;
  background: $color-neutral-100;
  border-radius: $border-radius-full;
  overflow: hidden;
  &__fill {
    height: 100%;
    background: $color-primary-500;
    border-radius: $border-radius-full;
    transition: width $transition-base;
  }
}

.chip-pulse {
  animation: chip-glow 1.5s ease-in-out infinite;
}

@keyframes chip-glow {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.retry-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.ack-details {
  &__error {
    margin-bottom: $space-4;
    h5 { font-size: $font-size-base; margin-bottom: $space-2; }
  }
}

.error-pre {
  font-family: $font-family-mono;
  font-size: $font-size-xs;
  background: $color-neutral-50;
  border: 1px solid $border-color;
  border-radius: $border-radius-base;
  padding: $space-3;
  max-height: 160px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

// Connectors
.connector-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: $space-4;
}

.connector-card {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: $space-4 $space-5;
    border-bottom: 1px solid $border-color;
    h4 { font-size: $font-size-md; margin: 0; text-transform: capitalize; }
  }
  &__health {
    font-size: $font-size-xs;
    font-weight: $font-weight-medium;
    padding: 2px 8px;
    border-radius: $border-radius-full;
    text-transform: capitalize;
    &--healthy { background: $color-success-50; color: $color-success-700; }
    &--degraded { background: $color-warning-50; color: $color-warning-700; }
    &--error, &--unhealthy { background: $color-danger-50; color: $color-danger-700; }
    &--unknown { background: $color-neutral-100; color: $color-neutral-500; }
  }
  &__body { padding: $space-4 $space-5; }
  &__caps { display: flex; flex-wrap: wrap; gap: $space-2; }
  &__actions {
    display: flex; gap: $space-2; padding: $space-3 $space-5; border-top: 1px solid $border-color;
  }
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: $space-4;
}

.form-toggle {
  .toggle-label {
    display: flex;
    align-items: center;
    gap: $space-2;
    font-size: $font-size-base;
    color: $color-neutral-700;
    cursor: pointer;
    input { accent-color: $color-primary-500; width: 18px; height: 18px; }
  }
}

.schema-pre {
  font-family: $font-family-mono;
  font-size: $font-size-sm;
  background: $color-neutral-50;
  border: 1px solid $border-color;
  border-radius: $border-radius-base;
  padding: $space-4;
  max-height: 400px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
