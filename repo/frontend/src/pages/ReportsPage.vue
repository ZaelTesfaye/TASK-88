<template>
  <div class="page-content reports-page">
    <AppToast ref="toast" />

    <div class="page-header">
      <h2 class="page-header__title">Reports</h2>
      <div class="page-header__actions">
        <AppButton
          v-if="activeTab === 'schedules'"
          variant="primary"
          @click="openNewSchedule"
          >New Schedule</AppButton
        >
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
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Schedules Tab -->
    <div v-if="activeTab === 'schedules'" class="tab-panel">
      <AppLoadingState v-if="schedulesLoading" message="Loading schedules..." />
      <AppErrorState
        v-else-if="schedulesError"
        :message="schedulesError"
        retryable
        @retry="loadSchedules"
      />
      <AppEmptyState
        v-else-if="schedules.length === 0"
        title="No schedules"
        description="Create a report schedule to automate report generation."
      >
        <template #action
          ><AppButton variant="primary" @click="openNewSchedule"
            >New Schedule</AppButton
          ></template
        >
      </AppEmptyState>

      <AppTable
        v-else
        :columns="scheduleColumns"
        :rows="schedules"
        :loading="schedulesLoading"
      >
        <template #cell-cron="{ row }">
          <div class="cron-cell">
            <code class="cron-code">{{ row.cronExpression }}</code>
            <span class="text-xs text-muted">{{
              cronToHuman(row.cronExpression)
            }}</span>
          </div>
        </template>
        <template #cell-format="{ row }">
          <AppChip
            :label="row.format"
            :variant="row.format === 'PDF' ? 'info' : 'neutral'"
            size="sm"
          />
        </template>
        <template #cell-enabled="{ row }">
          <label class="inline-toggle" @click.stop>
            <input
              type="checkbox"
              :checked="row.enabled"
              @change="toggleScheduleEnabled(row)"
            />
            <span class="inline-toggle__track" />
          </label>
        </template>
        <template #cell-missedPolicy="{ row }">
          <AppChip
            :label="row.missedRunPolicy || 'skip'"
            :variant="row.missedRunPolicy === 'run' ? 'warning' : 'neutral'"
            size="sm"
          />
        </template>
        <template #cell-actions="{ row }">
          <AppButton variant="ghost" size="sm" @click="openEditSchedule(row)"
            >Edit</AppButton
          >
        </template>
      </AppTable>
    </div>

    <!-- Run History Tab -->
    <div v-if="activeTab === 'history'" class="tab-panel">
      <!-- Filters -->
      <div class="history-filters">
        <AppSelect
          v-model="historyScheduleFilter"
          :options="scheduleFilterOptions"
          placeholder="All schedules"
        />
        <AppSelect
          v-model="historyStateFilter"
          :options="stateFilterOptions"
          placeholder="All states"
        />
        <AppInput v-model="historyDateFrom" type="date" placeholder="From" />
        <AppInput v-model="historyDateTo" type="date" placeholder="To" />
        <AppButton variant="outline" size="sm" @click="loadRuns"
          >Apply</AppButton
        >
      </div>

      <AppLoadingState v-if="runsLoading" message="Loading run history..." />
      <AppErrorState
        v-else-if="runsError"
        :message="runsError"
        retryable
        @retry="loadRuns"
      />
      <AppEmptyState
        v-else-if="runs.length === 0"
        title="No run history"
        description="Runs will appear here once schedules have executed."
      />

      <AppTable
        v-else
        :columns="runColumns"
        :rows="runs"
        :loading="runsLoading"
        :current-page="runPage"
        :total-pages="runTotalPages"
        :total-items="runTotalItems"
        @page-change="
          (p) => {
            runPage = p;
            loadRuns();
          }
        "
      >
        <template #cell-state="{ row }">
          <AppChip :status="row.state" :label="row.state" />
        </template>
        <template #cell-duration="{ row }">
          <span class="text-sm">{{ computeDuration(row) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <div class="run-actions">
            <AppButton
              v-if="row.state === 'ready'"
              variant="primary"
              size="sm"
              :loading="row._downloading"
              @click="downloadReport(row)"
              >Download</AppButton
            >
            <button
              v-if="row.state === 'failed'"
              class="error-toggle"
              @click="row._showError = !row._showError"
            >
              {{ row._showError ? "Hide Error" : "View Error" }}
            </button>
          </div>
          <div v-if="row._showError" class="run-error">
            <pre class="error-pre">{{
              row.error || row.errorDetails || "No error details"
            }}</pre>
          </div>
        </template>
      </AppTable>
    </div>

    <!-- Schedule Dialog (Add/Edit) -->
    <AppDialog
      v-model="showScheduleDialog"
      :title="editingSchedule ? 'Edit Schedule' : 'New Schedule'"
      size="lg"
      persistent
    >
      <div class="form-stack">
        <AppInput
          v-model="scheduleForm.name"
          label="Name"
          required
          :error="formErrors.name"
        />

        <div class="form-group">
          <AppInput
            v-model="scheduleForm.cronExpression"
            label="Cron Expression"
            required
            :error="formErrors.cronExpression"
            hint="e.g. 0 6 * * * (daily at 6am)"
          />
          <div class="cron-presets">
            <span class="text-xs text-muted">Presets:</span>
            <button
              class="preset-btn"
              @click="scheduleForm.cronExpression = '0 6 * * *'"
            >
              Daily 6am
            </button>
            <button
              class="preset-btn"
              @click="scheduleForm.cronExpression = '0 6 * * 1'"
            >
              Weekly Mon
            </button>
            <button
              class="preset-btn"
              @click="scheduleForm.cronExpression = '0 6 1 * *'"
            >
              Monthly 1st
            </button>
          </div>
        </div>

        <AppSelect
          v-model="scheduleForm.timezone"
          label="Timezone"
          :options="timezoneOptions"
          placeholder="Select timezone..."
        />

        <div class="form-group">
          <label class="form-label">Output Format</label>
          <div class="radio-group">
            <label class="radio-label">
              <input type="radio" v-model="scheduleForm.format" value="CSV" />
              <span>CSV</span>
            </label>
            <label class="radio-label">
              <input type="radio" v-model="scheduleForm.format" value="PDF" />
              <span>PDF</span>
            </label>
          </div>
        </div>

        <AppInput
          v-model="scheduleForm.scope"
          label="Scope"
          hint="Node ID or leave empty for current context"
        />

        <div class="form-toggle">
          <label class="toggle-label">
            <input type="checkbox" v-model="scheduleForm.enabled" />
            <span>Enabled</span>
          </label>
        </div>
      </div>
      <template #footer>
        <AppButton variant="outline" @click="showScheduleDialog = false"
          >Cancel</AppButton
        >
        <AppButton
          variant="primary"
          :loading="savingSchedule"
          @click="saveSchedule"
          >{{ editingSchedule ? "Update" : "Create" }}</AppButton
        >
      </template>
    </AppDialog>

    <!-- Enable/Disable Confirmation -->
    <AppDialog
      v-model="showToggleConfirm"
      title="Confirm Status Change"
      size="sm"
    >
      <p>
        Are you sure you want to
        <strong>{{ toggleTarget?.enabled ? "disable" : "enable" }}</strong> the
        schedule "{{ toggleTarget?.name }}"?
      </p>
      <template #footer>
        <AppButton variant="outline" @click="showToggleConfirm = false"
          >Cancel</AppButton
        >
        <AppButton
          variant="primary"
          :loading="toggling"
          @click="confirmToggleEnabled"
          >Confirm</AppButton
        >
      </template>
    </AppDialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from "vue";
import { useContextStore } from "@/stores/context.js";
import { useAuthStore } from "@/stores/auth.js";
import * as reportsApi from "@/api/reports.js";
import AppButton from "@/components/common/AppButton.vue";
import AppInput from "@/components/common/AppInput.vue";
import AppSelect from "@/components/common/AppSelect.vue";
import AppChip from "@/components/common/AppChip.vue";
import AppTable from "@/components/common/AppTable.vue";
import AppDialog from "@/components/common/AppDialog.vue";
import AppLoadingState from "@/components/common/AppLoadingState.vue";
import AppErrorState from "@/components/common/AppErrorState.vue";
import AppEmptyState from "@/components/common/AppEmptyState.vue";
import AppToast from "@/components/common/AppToast.vue";

const contextStore = useContextStore();
const authStore = useAuthStore();
const toast = ref(null);

const tabs = [
  { key: "schedules", label: "Schedules" },
  { key: "history", label: "Run History" },
];
const activeTab = ref("schedules");

// ---- Schedules ----
const schedules = ref([]);
const schedulesLoading = ref(false);
const schedulesError = ref("");

const scheduleColumns = [
  { key: "name", label: "Name", sortable: true },
  { key: "cron", label: "Schedule" },
  { key: "timezone", label: "Timezone" },
  { key: "format", label: "Format", width: "90px" },
  { key: "scope", label: "Scope" },
  { key: "enabled", label: "Enabled", width: "80px" },
  { key: "missedPolicy", label: "Missed Policy", width: "110px" },
  { key: "actions", label: "", width: "80px" },
];

async function loadSchedules() {
  schedulesLoading.value = true;
  schedulesError.value = "";
  try {
    const { data } = await reportsApi.getSchedules();
    schedules.value = Array.isArray(data) ? data : data.items || [];
  } catch (err) {
    schedulesError.value = err.message || "Failed to load schedules";
  } finally {
    schedulesLoading.value = false;
  }
}

// Schedule form
const showScheduleDialog = ref(false);
const editingSchedule = ref(null);
const savingSchedule = ref(false);
const scheduleForm = ref({
  name: "",
  cronExpression: "",
  timezone: "UTC",
  format: "CSV",
  scope: "",
  enabled: true,
});
const formErrors = ref({});

const timezoneOptions = [
  "UTC",
  "US/Eastern",
  "US/Central",
  "US/Mountain",
  "US/Pacific",
  "Europe/London",
  "Europe/Berlin",
  "Asia/Tokyo",
  "Asia/Shanghai",
  "Australia/Sydney",
];

function openNewSchedule() {
  editingSchedule.value = null;
  scheduleForm.value = {
    name: "",
    cronExpression: "",
    timezone: "UTC",
    format: "CSV",
    scope: contextStore.currentNode?.id || "",
    enabled: true,
  };
  formErrors.value = {};
  showScheduleDialog.value = true;
}

function openEditSchedule(sched) {
  editingSchedule.value = sched;
  scheduleForm.value = {
    name: sched.name || "",
    cronExpression: sched.cronExpression || sched.cron || "",
    timezone: sched.timezone || "UTC",
    format: sched.format || "CSV",
    scope: sched.scope || "",
    enabled: sched.enabled !== false,
  };
  formErrors.value = {};
  showScheduleDialog.value = true;
}

function validateScheduleForm() {
  const e = {};
  if (!scheduleForm.value.name.trim()) e.name = "Name is required";
  if (!scheduleForm.value.cronExpression.trim())
    e.cronExpression = "Cron expression is required";
  formErrors.value = e;
  return Object.keys(e).length === 0;
}

async function saveSchedule() {
  if (!validateScheduleForm()) return;
  savingSchedule.value = true;
  try {
    if (editingSchedule.value) {
      await reportsApi.updateSchedule(
        editingSchedule.value.id,
        scheduleForm.value,
      );
      toast.value?.addToast({ message: "Schedule updated", type: "success" });
    } else {
      await reportsApi.createSchedule(scheduleForm.value);
      toast.value?.addToast({ message: "Schedule created", type: "success" });
    }
    showScheduleDialog.value = false;
    await loadSchedules();
  } catch (err) {
    toast.value?.addToast({
      message: err.message || "Save failed",
      type: "error",
    });
  } finally {
    savingSchedule.value = false;
  }
}

// Enable/disable toggle
const showToggleConfirm = ref(false);
const toggleTarget = ref(null);
const toggling = ref(false);

function toggleScheduleEnabled(sched) {
  toggleTarget.value = sched;
  showToggleConfirm.value = true;
}

async function confirmToggleEnabled() {
  toggling.value = true;
  try {
    await reportsApi.updateSchedule(toggleTarget.value.id, {
      enabled: !toggleTarget.value.enabled,
    });
    toggleTarget.value.enabled = !toggleTarget.value.enabled;
    toast.value?.addToast({
      message: `Schedule ${toggleTarget.value.enabled ? "enabled" : "disabled"}`,
      type: "success",
    });
    showToggleConfirm.value = false;
  } catch (err) {
    toast.value?.addToast({
      message: err.message || "Toggle failed",
      type: "error",
    });
  } finally {
    toggling.value = false;
  }
}

// Cron to human-readable
function cronToHuman(cron) {
  if (!cron) return "";
  const parts = cron.trim().split(/\s+/);
  if (parts.length < 5) return cron;
  const [min, hour, dom, , dow] = parts;
  if (dom === "1" && dow === "*")
    return `Monthly on the 1st at ${hour}:${min.padStart(2, "0")}`;
  if (dow === "1") return `Weekly on Monday at ${hour}:${min.padStart(2, "0")}`;
  if (dom === "*" && dow === "*")
    return `Daily at ${hour}:${min.padStart(2, "0")}`;
  return cron;
}

// ---- Run History ----
const runs = ref([]);
const runsLoading = ref(false);
const runsError = ref("");
const runPage = ref(1);
const runTotalPages = ref(1);
const runTotalItems = ref(0);
const historyScheduleFilter = ref("");
const historyStateFilter = ref("");
const historyDateFrom = ref("");
const historyDateTo = ref("");

const scheduleFilterOptions = computed(() => [
  { value: "", label: "All schedules" },
  ...schedules.value.map((s) => ({ value: s.id, label: s.name })),
]);

const stateFilterOptions = [
  { value: "", label: "All states" },
  { value: "ready", label: "Ready" },
  { value: "failed", label: "Failed" },
];

const runColumns = [
  { key: "id", label: "ID", width: "80px" },
  { key: "scheduleName", label: "Schedule" },
  { key: "state", label: "State", width: "100px" },
  { key: "started_at", label: "Started" },
  { key: "finished_at", label: "Finished" },
  { key: "duration", label: "Duration", width: "100px" },
  { key: "requested_by", label: "Requested By" },
  { key: "actions", label: "", width: "160px" },
];

async function loadRuns() {
  runsLoading.value = true;
  runsError.value = "";
  try {
    const params = { page: runPage.value };
    if (historyStateFilter.value) params.state = historyStateFilter.value;
    if (historyDateFrom.value) params.date_from = historyDateFrom.value;
    if (historyDateTo.value) params.date_to = historyDateTo.value;

    const schedId = historyScheduleFilter.value || schedules.value[0]?.id;
    if (!schedId) {
      runs.value = [];
      runsLoading.value = false;
      return;
    }
    params.schedule_id = schedId;
    const { data } = await reportsApi.getRuns(params);
    const list = Array.isArray(data) ? data : data.items || [];
    runs.value = list.map((r) => ({
      ...r,
      _showError: false,
      _downloading: false,
    }));
    runTotalPages.value = data.totalPages || 1;
    runTotalItems.value = data.totalItems || list.length;
  } catch (err) {
    runsError.value = err.message || "Failed to load runs";
  } finally {
    runsLoading.value = false;
  }
}

function computeDuration(row) {
  if (!row.started_at || !row.finished_at) return "--";
  const diff = new Date(row.finished_at) - new Date(row.started_at);
  if (diff < 1000) return `${diff}ms`;
  if (diff < 60000) return `${(diff / 1000).toFixed(1)}s`;
  return `${Math.floor(diff / 60000)}m ${Math.round((diff % 60000) / 1000)}s`;
}

async function downloadReport(row) {
  row._downloading = true;
  try {
    // Access check first
    const { data: accessData } = await reportsApi.checkAccess(row.id);
    const denied =
      accessData?.has_access === false || accessData?.allowed === false;
    if (denied) {
      toast.value?.addToast({
        message: "You no longer have permission to download this report.",
        type: "error",
      });
      return;
    }
    const { data } = await reportsApi.downloadRun(row.id);
    const url = URL.createObjectURL(data);
    const a = document.createElement("a");
    a.href = url;
    a.download = `report-${row.id}.${row.format?.toLowerCase() || "csv"}`;
    a.click();
    URL.revokeObjectURL(url);
    toast.value?.addToast({ message: "Download started", type: "success" });
  } catch (err) {
    toast.value?.addToast({
      message: err.message || "Download failed",
      type: "error",
    });
  } finally {
    row._downloading = false;
  }
}

// ---- Tab switching ----
watch(activeTab, (tab) => {
  if (tab === "schedules" && schedules.value.length === 0) loadSchedules();
  if (tab === "history") loadRuns();
});

onMounted(() => {
  loadSchedules();
});
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
  &:hover {
    color: $color-neutral-700;
  }
  &--active {
    color: $color-primary-600;
    border-bottom-color: $color-primary-500;
  }
}

.tab-panel {
  min-height: 300px;
}

// Cron cell
.cron-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cron-code {
  font-family: $font-family-mono;
  font-size: $font-size-sm;
  background: $color-neutral-50;
  padding: 2px 6px;
  border-radius: $border-radius-sm;
  color: $color-neutral-700;
}

// Inline toggle switch
.inline-toggle {
  display: inline-flex;
  align-items: center;
  cursor: pointer;

  input {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
  }

  &__track {
    display: block;
    width: 36px;
    height: 20px;
    border-radius: $border-radius-full;
    background: $color-neutral-300;
    position: relative;
    transition: background $transition-fast;

    &::after {
      content: "";
      position: absolute;
      top: 2px;
      left: 2px;
      width: 16px;
      height: 16px;
      border-radius: 50%;
      background: #fff;
      box-shadow: $shadow-xs;
      transition: transform $transition-fast;
    }
  }

  input:checked + &__track {
    background: $color-primary-500;
    &::after {
      transform: translateX(16px);
    }
  }
}

// History filters
.history-filters {
  display: flex;
  gap: $space-3;
  flex-wrap: wrap;
  margin-bottom: $space-5;
  align-items: flex-end;
}

// Run actions
.run-actions {
  display: flex;
  gap: $space-2;
  align-items: center;
}

.error-toggle {
  font-size: $font-size-xs;
  color: $color-danger-500;
  font-weight: $font-weight-medium;
  cursor: pointer;
  &:hover {
    text-decoration: underline;
  }
}

.run-error {
  margin-top: $space-2;
}

.error-pre {
  font-family: $font-family-mono;
  font-size: $font-size-xs;
  background: $color-danger-50;
  border: 1px solid $color-danger-100;
  border-radius: $border-radius-base;
  padding: $space-3;
  max-height: 120px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  color: $color-danger-700;
}

// Form
.form-stack {
  display: flex;
  flex-direction: column;
  gap: $space-4;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: $space-2;
}

.form-label {
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  color: $color-neutral-700;
}

.cron-presets {
  display: flex;
  align-items: center;
  gap: $space-2;
}

.preset-btn {
  font-size: $font-size-xs;
  padding: 2px 8px;
  border-radius: $border-radius-base;
  border: 1px solid $border-color;
  background: $color-neutral-0;
  color: $color-primary-600;
  cursor: pointer;
  transition: all $transition-fast;
  &:hover {
    background: $color-primary-50;
    border-color: $color-primary-300;
  }
}

.radio-group {
  display: flex;
  gap: $space-4;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: $space-2;
  font-size: $font-size-base;
  color: $color-neutral-700;
  cursor: pointer;
  input {
    accent-color: $color-primary-500;
  }
}

.form-toggle {
  .toggle-label {
    display: flex;
    align-items: center;
    gap: $space-2;
    font-size: $font-size-base;
    color: $color-neutral-700;
    cursor: pointer;
    input {
      accent-color: $color-primary-500;
      width: 18px;
      height: 18px;
    }
  }
}
</style>
