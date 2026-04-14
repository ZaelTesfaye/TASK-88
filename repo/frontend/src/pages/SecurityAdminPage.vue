<template>
  <div class="page-content security-page">
    <AppToast ref="toast" />

    <div class="page-header">
      <h2 class="page-header__title">Security Administration</h2>
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

    <!-- Sensitive Fields Tab -->
    <div v-if="activeTab === 'fields'" class="tab-panel">
      <AppLoadingState v-if="fieldsLoading" message="Loading sensitive fields..." />
      <AppErrorState v-else-if="fieldsError" :message="fieldsError" retryable @retry="loadFields" />
      <AppEmptyState v-else-if="fields.length === 0" title="No sensitive fields" description="No sensitive field configurations found." />

      <AppTable v-else :columns="fieldColumns" :rows="fields" :loading="fieldsLoading">
        <template #cell-mask_pattern="{ row }">
          <code class="mono-code">{{ row.mask_pattern }}</code>
        </template>
        <template #cell-unmask_roles="{ row }">
          <div class="chip-list">
            <AppChip v-for="role in (row.unmask_roles || [])" :key="role" :label="role" variant="info" size="sm" />
          </div>
        </template>
        <template #cell-preview="{ row }">
          <span class="masked-preview">{{ previewMask(row) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <AppButton variant="ghost" size="sm" @click="openEditField(row)">Edit</AppButton>
        </template>
      </AppTable>
    </div>

    <!-- Retention Tab -->
    <div v-if="activeTab === 'retention'" class="tab-panel">
      <AppLoadingState v-if="retentionLoading" message="Loading retention policies..." />
      <AppErrorState v-else-if="retentionError" :message="retentionError" retryable @retry="loadRetention" />
      <AppEmptyState v-else-if="retentionPolicies.length === 0" title="No retention policies" description="No artifact retention policies configured." />

      <AppTable v-else :columns="retentionColumns" :rows="retentionPolicies" :loading="retentionLoading">
        <template #cell-retention_days="{ row }">
          <div class="inline-edit" v-if="editingRetentionId === row.id">
            <AppInput v-model="editRetentionDays" type="number" min="1" />
            <AppButton variant="primary" size="sm" @click="saveRetention(row)">Save</AppButton>
            <AppButton variant="ghost" size="sm" @click="editingRetentionId = null">Cancel</AppButton>
          </div>
          <span v-else class="editable-value" @click="startEditRetention(row)">{{ row.retention_days }} days</span>
        </template>
        <template #cell-legal_hold="{ row }">
          <label class="inline-toggle" @click.stop>
            <input type="checkbox" :checked="row.legal_hold_enabled" @change="toggleLegalHoldPolicy(row)" />
            <span class="inline-toggle__track" />
          </label>
        </template>
        <template #cell-last_updated="{ row }">
          <span class="text-sm">{{ row.last_updated ? new Date(row.last_updated).toLocaleDateString() : '--' }}</span>
        </template>
      </AppTable>
    </div>

    <!-- Legal Holds Tab -->
    <div v-if="activeTab === 'holds'" class="tab-panel">
      <div class="tab-panel__actions">
        <AppButton variant="primary" @click="openCreateHold">Create Legal Hold</AppButton>
        <AppButton variant="outline" :loading="dryRunning" @click="runPurgeDryRun">Purge Dry Run</AppButton>
        <AppButton variant="danger" :disabled="!purgePreview" @click="showPurgeConfirm = true">Execute Purge</AppButton>
      </div>

      <AppLoadingState v-if="holdsLoading" message="Loading legal holds..." />
      <AppErrorState v-else-if="holdsError" :message="holdsError" retryable @retry="loadHolds" />
      <AppEmptyState v-else-if="holds.length === 0" title="No active legal holds" description="Create a legal hold to prevent data purging." />

      <AppTable v-else :columns="holdsColumns" :rows="holds" :loading="holdsLoading">
        <template #cell-scope="{ row }">
          <code class="mono-code">{{ typeof row.scope === 'object' ? JSON.stringify(row.scope) : row.scope }}</code>
        </template>
        <template #cell-actions="{ row }">
          <AppButton variant="danger" size="sm" @click="openReleaseHold(row)">Release</AppButton>
        </template>
      </AppTable>

      <!-- Dry run preview -->
      <div v-if="purgePreview" class="purge-preview card mt-4">
        <div class="card__header">
          <span class="card__title">Purge Preview</span>
        </div>
        <div class="card__body">
          <p class="text-sm">The following items would be purged:</p>
          <ul class="purge-list">
            <li v-for="(item, idx) in purgePreview.items || []" :key="idx" class="text-sm">{{ item.type }}: {{ item.count }} records</li>
          </ul>
          <p class="text-sm text-muted mt-2">Total: {{ purgePreview.totalCount || 0 }} records</p>
        </div>
      </div>
    </div>

    <!-- Audit Deletion Tab -->
    <div v-if="activeTab === 'audit'" class="tab-panel">
      <AppLoadingState v-if="auditLoading" message="Loading deletion requests..." />
      <AppErrorState v-else-if="auditError" :message="auditError" retryable @retry="loadAuditRequests" />
      <AppEmptyState v-else-if="auditRequests.length === 0" title="No deletion requests" description="No pending audit deletion requests." />

      <AppTable v-else :columns="auditColumns" :rows="auditRequests" :loading="auditLoading">
        <template #cell-state="{ row }">
          <AppChip :status="row.state" :label="row.state" />
        </template>
        <template #cell-approvals="{ row }">
          <div class="approval-indicator">
            <div class="approval-dots">
              <span class="approval-dot" :class="{ 'approval-dot--filled': (row.approvals || 0) >= 1 }" />
              <span class="approval-dot" :class="{ 'approval-dot--filled': (row.approvals || 0) >= 2 }" />
            </div>
            <span class="text-xs">{{ row.approvals || 0 }}/2 approved</span>
          </div>
        </template>
        <template #cell-actions="{ row }">
          <div class="flex gap-2">
            <AppButton
              v-if="row.state === 'pending' && !hasUserApproved(row)"
              variant="success"
              size="sm"
              @click="approveAuditDeletion(row)"
            >Approve</AppButton>
            <AppButton
              v-if="row.state === 'pending'"
              variant="outline"
              size="sm"
              @click="openRejectDialog(row)"
            >Reject</AppButton>
            <span v-if="hasUserApproved(row)" class="text-xs text-success">You approved</span>
          </div>
        </template>
      </AppTable>
    </div>

    <!-- Password Resets Tab -->
    <div v-if="activeTab === 'passwords'" class="tab-panel">
      <div class="tab-panel__actions">
        <AppButton variant="primary" @click="showNewResetDialog = true">New Reset Request</AppButton>
      </div>

      <AppLoadingState v-if="passwordLoading" message="Loading reset requests..." />
      <AppErrorState v-else-if="passwordError" :message="passwordError" retryable @retry="loadPasswordResets" />
      <AppEmptyState v-else-if="passwordResets.length === 0" title="No reset requests" description="No password reset requests found." />

      <AppTable v-else :columns="passwordColumns" :rows="passwordResets" :loading="passwordLoading">
        <template #cell-state="{ row }">
          <AppChip :status="row.state" :label="row.state" />
        </template>
        <template #cell-actions="{ row }">
          <AppButton
            v-if="row.state === 'pending'"
            variant="primary"
            size="sm"
            :loading="row._approving"
            @click="approvePasswordReset(row)"
          >Approve</AppButton>
          <div v-if="row._token" class="token-display">
            <code class="mono-code token-code">{{ row._token }}</code>
            <span class="text-xs text-warning">One-time token -- copy now</span>
          </div>
        </template>
      </AppTable>
    </div>

    <!-- Key Rotation Tab -->
    <div v-if="activeTab === 'keys'" class="tab-panel">
      <AppLoadingState v-if="keysLoading" message="Loading key ring..." />
      <AppErrorState v-else-if="keysError" :message="keysError" retryable @retry="loadKeys" />
      <AppEmptyState v-else-if="keys.length === 0" title="No encryption keys" description="No key ring entries found." />

      <AppTable v-else :columns="keyColumns" :rows="keys" :loading="keysLoading">
        <template #cell-status="{ row }">
          <AppChip :status="row.status" :label="row.status" />
        </template>
        <template #cell-rotates_at="{ row }">
          <div class="rotation-info">
            <span class="text-sm">{{ row.rotates_at ? new Date(row.rotates_at).toLocaleDateString() : '--' }}</span>
            <span v-if="daysUntilRotation(row) !== null" class="text-xs" :class="daysUntilRotation(row) <= 7 ? 'text-warning' : 'text-muted'">
              {{ daysUntilRotation(row) }} days
            </span>
          </div>
        </template>
        <template #cell-actions="{ row }">
          <AppButton variant="outline" size="sm" @click="openRotateConfirm(row)">Rotate Now</AppButton>
        </template>
      </AppTable>

      <!-- Rotation history -->
      <div v-if="rotationHistory.length > 0" class="rotation-history mt-6">
        <h4 class="mb-3">Rotation History</h4>
        <div class="history-list">
          <div v-for="entry in rotationHistory" :key="entry.id" class="history-entry">
            <span class="text-sm font-medium">{{ entry.key_id }}</span>
            <span class="text-xs text-muted">{{ new Date(entry.rotated_at).toLocaleString() }}</span>
            <span class="text-xs">{{ entry.rotated_by }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit Sensitive Field Dialog -->
    <AppDialog v-model="showFieldDialog" title="Edit Sensitive Field" size="md" persistent>
      <div class="form-stack">
        <AppInput :modelValue="editingField?.field_key" label="Field Key" disabled />
        <AppInput v-model="fieldForm.mask_pattern" label="Mask Pattern" hint="e.g. ***-**-{last4}" />
        <AppInput v-model="fieldForm.unmask_roles" label="Unmask Roles" hint="Comma-separated roles" />
      </div>
      <template #footer>
        <AppButton variant="outline" @click="showFieldDialog = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="savingField" @click="saveField">Save</AppButton>
      </template>
    </AppDialog>

    <!-- Create Legal Hold Dialog -->
    <AppDialog v-model="showHoldDialog" title="Create Legal Hold" size="md" persistent>
      <div class="form-stack">
        <AppInput v-model="holdForm.scope" label="Scope (JSON)" type="textarea" :rows="3" required hint='e.g. {"nodeId": "abc", "type": "department"}' :error="holdFormError" />
        <AppInput v-model="holdForm.reason" label="Reason" type="textarea" :rows="2" required />
      </div>
      <template #footer>
        <AppButton variant="outline" @click="showHoldDialog = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="creatingHold" @click="createHold">Create</AppButton>
      </template>
    </AppDialog>

    <!-- Release Hold Confirmation -->
    <AppDialog v-model="showReleaseConfirm" title="Release Legal Hold" size="sm" danger>
      <p>Are you sure you want to release this legal hold? Data covered by this hold will become eligible for purging.</p>
      <template #footer>
        <AppButton variant="outline" @click="showReleaseConfirm = false">Cancel</AppButton>
        <AppButton variant="danger" :loading="releasingHold" @click="releaseHold">Release Hold</AppButton>
      </template>
    </AppDialog>

    <!-- Purge Confirmation -->
    <AppDialog v-model="showPurgeConfirm" title="Execute Purge" size="md" danger>
      <div class="purge-warning">
        <svg width="40" height="40" viewBox="0 0 40 40" fill="none"><path d="M20 4L2 36h36L20 4z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/><path d="M20 16v8M20 28v.5" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
        <p><strong>This action is irreversible.</strong> All records matching the purge criteria that are not under legal hold will be permanently deleted.</p>
        <p class="text-sm text-muted">Total records to purge: {{ purgePreview?.totalCount || 0 }}</p>
      </div>
      <template #footer>
        <AppButton variant="outline" @click="showPurgeConfirm = false">Cancel</AppButton>
        <AppButton variant="danger" :loading="purging" @click="executePurge">Confirm Purge</AppButton>
      </template>
    </AppDialog>

    <!-- Reject Audit Dialog -->
    <AppDialog v-model="showRejectDialog" title="Reject Deletion Request" size="sm">
      <AppInput v-model="rejectReason" label="Reason" type="textarea" :rows="3" required :error="rejectReasonError" />
      <template #footer>
        <AppButton variant="outline" @click="showRejectDialog = false">Cancel</AppButton>
        <AppButton variant="danger" :loading="rejecting" @click="rejectAuditDeletion">Reject</AppButton>
      </template>
    </AppDialog>

    <!-- Rotate Key Confirmation -->
    <AppDialog v-model="showRotateConfirm" title="Rotate Encryption Key" size="sm" danger>
      <p>Rotate key <strong>{{ rotatingKey?.key_id }}</strong> immediately? All new data will be encrypted with the new key. Existing data will be re-encrypted during the next migration cycle.</p>
      <template #footer>
        <AppButton variant="outline" @click="showRotateConfirm = false">Cancel</AppButton>
        <AppButton variant="danger" :loading="rotating" @click="rotateKey">Rotate Now</AppButton>
      </template>
    </AppDialog>

    <!-- New Password Reset Dialog -->
    <AppDialog v-model="showNewResetDialog" title="New Password Reset" size="md">
      <div class="form-stack">
        <AppInput v-model="newResetForm.userId" label="User ID" required />
        <AppInput v-model="newResetForm.reason" label="Reason" type="textarea" :rows="2" />
      </div>
      <template #footer>
        <AppButton variant="outline" @click="showNewResetDialog = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="creatingReset" @click="createPasswordReset">Submit</AppButton>
      </template>
    </AppDialog>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue';
import { useAuthStore } from '@/stores/auth.js';
import * as securityApi from '@/api/security.js';
import * as auditApi from '@/api/audit.js';
import AppButton from '@/components/common/AppButton.vue';
import AppInput from '@/components/common/AppInput.vue';
import AppChip from '@/components/common/AppChip.vue';
import AppTable from '@/components/common/AppTable.vue';
import AppDialog from '@/components/common/AppDialog.vue';
import AppLoadingState from '@/components/common/AppLoadingState.vue';
import AppErrorState from '@/components/common/AppErrorState.vue';
import AppEmptyState from '@/components/common/AppEmptyState.vue';
import AppToast from '@/components/common/AppToast.vue';

const authStore = useAuthStore();
const toast = ref(null);

const tabs = [
  { key: 'fields', label: 'Sensitive Fields' },
  { key: 'retention', label: 'Retention' },
  { key: 'holds', label: 'Legal Holds' },
  { key: 'audit', label: 'Audit Deletion' },
  { key: 'passwords', label: 'Password Resets' },
  { key: 'keys', label: 'Key Rotation' },
];
const activeTab = ref('fields');

// ==============================
// Sensitive Fields
// ==============================
const fields = ref([]);
const fieldsLoading = ref(false);
const fieldsError = ref('');
const showFieldDialog = ref(false);
const editingField = ref(null);
const savingField = ref(false);
const fieldForm = ref({ mask_pattern: '', unmask_roles: '' });

const fieldColumns = [
  { key: 'field_key', label: 'Field Key' },
  { key: 'mask_pattern', label: 'Mask Pattern' },
  { key: 'unmask_roles', label: 'Unmask Roles' },
  { key: 'preview', label: 'Preview' },
  { key: 'actions', label: '', width: '80px' },
];

async function loadFields() {
  fieldsLoading.value = true;
  fieldsError.value = '';
  try {
    const { data } = await securityApi.getSensitiveFields();
    fields.value = Array.isArray(data) ? data : data.items || [];
  } catch (err) {
    fieldsError.value = err.message || 'Failed to load fields';
  } finally {
    fieldsLoading.value = false;
  }
}

function previewMask(row) {
  const pattern = row.mask_pattern || '****';
  return pattern.replace(/\{[^}]+\}/g, '1234');
}

function openEditField(row) {
  editingField.value = row;
  fieldForm.value = {
    mask_pattern: row.mask_pattern || '',
    unmask_roles: Array.isArray(row.unmask_roles) ? row.unmask_roles.join(', ') : row.unmask_roles || '',
  };
  showFieldDialog.value = true;
}

async function saveField() {
  savingField.value = true;
  try {
    const roles = fieldForm.value.unmask_roles.split(',').map(r => r.trim()).filter(Boolean);
    await securityApi.updateSensitiveFields({
      field_key: editingField.value.field_key,
      mask_pattern: fieldForm.value.mask_pattern,
      unmask_roles: roles,
    });
    toast.value?.addToast({ message: 'Field updated', type: 'success' });
    showFieldDialog.value = false;
    await loadFields();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Update failed', type: 'error' });
  } finally {
    savingField.value = false;
  }
}

// ==============================
// Retention
// ==============================
const retentionPolicies = ref([]);
const retentionLoading = ref(false);
const retentionError = ref('');
const editingRetentionId = ref(null);
const editRetentionDays = ref('');

const retentionColumns = [
  { key: 'artifact_type', label: 'Artifact Type' },
  { key: 'retention_days', label: 'Retention Days' },
  { key: 'legal_hold', label: 'Legal Hold' },
  { key: 'last_updated', label: 'Last Updated' },
];

async function loadRetention() {
  retentionLoading.value = true;
  retentionError.value = '';
  try {
    const { data } = await securityApi.getRetentionPolicies();
    retentionPolicies.value = Array.isArray(data) ? data : data.items || [];
  } catch (err) {
    retentionError.value = err.message || 'Failed to load retention policies';
  } finally {
    retentionLoading.value = false;
  }
}

function startEditRetention(row) {
  editingRetentionId.value = row.id;
  editRetentionDays.value = String(row.retention_days);
}

async function saveRetention(row) {
  try {
    await securityApi.updateRetentionPolicies(row.id, { retention_days: parseInt(editRetentionDays.value, 10) });
    row.retention_days = parseInt(editRetentionDays.value, 10);
    editingRetentionId.value = null;
    toast.value?.addToast({ message: 'Retention updated', type: 'success' });
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Update failed', type: 'error' });
  }
}

async function toggleLegalHoldPolicy(row) {
  try {
    await securityApi.updateRetentionPolicies(row.id, { legal_hold_enabled: !row.legal_hold_enabled });
    row.legal_hold_enabled = !row.legal_hold_enabled;
    toast.value?.addToast({ message: 'Legal hold toggled', type: 'success' });
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Toggle failed', type: 'error' });
  }
}

// ==============================
// Legal Holds
// ==============================
const holds = ref([]);
const holdsLoading = ref(false);
const holdsError = ref('');
const showHoldDialog = ref(false);
const holdForm = ref({ scope: '', reason: '' });
const holdFormError = ref('');
const creatingHold = ref(false);
const showReleaseConfirm = ref(false);
const releaseTarget = ref(null);
const releasingHold = ref(false);
const purgePreview = ref(null);
const dryRunning = ref(false);
const showPurgeConfirm = ref(false);
const purging = ref(false);

const holdsColumns = [
  { key: 'scope', label: 'Scope' },
  { key: 'reason', label: 'Reason' },
  { key: 'created_by', label: 'Created By' },
  { key: 'created_at', label: 'Created' },
  { key: 'actions', label: '', width: '100px' },
];

async function loadHolds() {
  holdsLoading.value = true;
  holdsError.value = '';
  try {
    const { data } = await securityApi.getRetentionPolicies({ type: 'legal_holds' });
    holds.value = Array.isArray(data) ? data : data.holds || data.items || [];
  } catch (err) {
    holdsError.value = err.message || 'Failed to load legal holds';
  } finally {
    holdsLoading.value = false;
  }
}

function openCreateHold() {
  holdForm.value = { scope: '', reason: '' };
  holdFormError.value = '';
  showHoldDialog.value = true;
}

async function createHold() {
  if (!holdForm.value.scope.trim() || !holdForm.value.reason.trim()) {
    holdFormError.value = 'Scope and reason are required';
    return;
  }
  let scopeObj;
  try { scopeObj = JSON.parse(holdForm.value.scope); } catch { holdFormError.value = 'Scope must be valid JSON'; return; }
  creatingHold.value = true;
  try {
    await securityApi.createLegalHold({ scope: scopeObj, reason: holdForm.value.reason });
    toast.value?.addToast({ message: 'Legal hold created', type: 'success' });
    showHoldDialog.value = false;
    await loadHolds();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Creation failed', type: 'error' });
  } finally {
    creatingHold.value = false;
  }
}

function openReleaseHold(hold) {
  releaseTarget.value = hold;
  showReleaseConfirm.value = true;
}

async function releaseHold() {
  releasingHold.value = true;
  try {
    // Using update to release hold
    await securityApi.updateRetentionPolicies(releaseTarget.value.id, { released: true });
    toast.value?.addToast({ message: 'Legal hold released', type: 'success' });
    showReleaseConfirm.value = false;
    await loadHolds();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Release failed', type: 'error' });
  } finally {
    releasingHold.value = false;
  }
}

async function runPurgeDryRun() {
  dryRunning.value = true;
  try {
    const { data } = await securityApi.dryRunPurge();
    purgePreview.value = data;
    toast.value?.addToast({ message: 'Dry run complete', type: 'info' });
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Dry run failed', type: 'error' });
  } finally {
    dryRunning.value = false;
  }
}

async function executePurge() {
  purging.value = true;
  try {
    await securityApi.executePurge();
    toast.value?.addToast({ message: 'Purge executed successfully', type: 'success' });
    showPurgeConfirm.value = false;
    purgePreview.value = null;
    await loadHolds();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Purge failed', type: 'error' });
  } finally {
    purging.value = false;
  }
}

// ==============================
// Audit Deletion
// ==============================
const auditRequests = ref([]);
const auditLoading = ref(false);
const auditError = ref('');
const showRejectDialog = ref(false);
const rejectTarget = ref(null);
const rejectReason = ref('');
const rejectReasonError = ref('');
const rejecting = ref(false);

const auditColumns = [
  { key: 'id', label: 'ID', width: '80px' },
  { key: 'requested_by', label: 'Requested By' },
  { key: 'reason', label: 'Reason' },
  { key: 'state', label: 'State' },
  { key: 'approvals', label: 'Approvals' },
  { key: 'actions', label: '', width: '180px' },
];

async function loadAuditRequests() {
  auditLoading.value = true;
  auditError.value = '';
  try {
    const { data } = await auditApi.getLogs({ type: 'delete_requests' });
    auditRequests.value = Array.isArray(data) ? data : data.items || data.requests || [];
  } catch (err) {
    auditError.value = err.message || 'Failed to load audit requests';
  } finally {
    auditLoading.value = false;
  }
}

function hasUserApproved(row) {
  return (row.approvedBy || []).includes(authStore.user?.id);
}

async function approveAuditDeletion(row) {
  try {
    await auditApi.approveDeleteRequest(row.id);
    row.approvals = (row.approvals || 0) + 1;
    row.approvedBy = [...(row.approvedBy || []), authStore.user?.id];
    if (row.approvals >= 2) row.state = 'approved';
    toast.value?.addToast({ message: 'Deletion request approved', type: 'success' });
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Approval failed', type: 'error' });
  }
}

function openRejectDialog(row) {
  rejectTarget.value = row;
  rejectReason.value = '';
  rejectReasonError.value = '';
  showRejectDialog.value = true;
}

async function rejectAuditDeletion() {
  if (!rejectReason.value.trim()) { rejectReasonError.value = 'Reason is required'; return; }
  rejecting.value = true;
  try {
    // Rejection via the same endpoint with reject action
    await auditApi.approveDeleteRequest(rejectTarget.value.id, { action: 'reject', reason: rejectReason.value });
    rejectTarget.value.state = 'rejected';
    toast.value?.addToast({ message: 'Request rejected', type: 'success' });
    showRejectDialog.value = false;
    await loadAuditRequests();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Rejection failed', type: 'error' });
  } finally {
    rejecting.value = false;
  }
}

// ==============================
// Password Resets
// ==============================
const passwordResets = ref([]);
const passwordLoading = ref(false);
const passwordError = ref('');
const showNewResetDialog = ref(false);
const newResetForm = ref({ userId: '', reason: '' });
const creatingReset = ref(false);

const passwordColumns = [
  { key: 'user', label: 'User' },
  { key: 'requested_by', label: 'Requested By' },
  { key: 'state', label: 'State' },
  { key: 'created_at', label: 'Created' },
  { key: 'actions', label: '', width: '200px' },
];

async function loadPasswordResets() {
  passwordLoading.value = true;
  passwordError.value = '';
  try {
    const { data } = await securityApi.getSensitiveFields({ type: 'password_resets' });
    passwordResets.value = (Array.isArray(data) ? data : data.items || data.requests || []).map(r => ({ ...r, _approving: false, _token: null }));
  } catch (err) {
    passwordError.value = err.message || 'Failed to load password resets';
  } finally {
    passwordLoading.value = false;
  }
}

async function createPasswordReset() {
  if (!newResetForm.value.userId) return;
  creatingReset.value = true;
  try {
    await securityApi.createPasswordResetRequest(newResetForm.value.userId);
    toast.value?.addToast({ message: 'Reset request created', type: 'success' });
    showNewResetDialog.value = false;
    await loadPasswordResets();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Request failed', type: 'error' });
  } finally {
    creatingReset.value = false;
  }
}

async function approvePasswordReset(row) {
  row._approving = true;
  try {
    const { data } = await securityApi.approvePasswordResetRequest(row.id);
    row.state = 'approved';
    row._token = data.token || data.oneTimeToken || 'TOKEN_UNAVAILABLE';
    toast.value?.addToast({ message: 'Reset approved. Copy the one-time token now.', type: 'success', duration: 8000 });
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Approval failed', type: 'error' });
  } finally {
    row._approving = false;
  }
}

// ==============================
// Key Rotation
// ==============================
const keys = ref([]);
const keysLoading = ref(false);
const keysError = ref('');
const rotationHistory = ref([]);
const showRotateConfirm = ref(false);
const rotatingKey = ref(null);
const rotating = ref(false);

const keyColumns = [
  { key: 'key_id', label: 'Key ID' },
  { key: 'purpose', label: 'Purpose' },
  { key: 'created_at', label: 'Created' },
  { key: 'rotates_at', label: 'Rotates At' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '', width: '120px' },
];

async function loadKeys() {
  keysLoading.value = true;
  keysError.value = '';
  try {
    const { data } = await securityApi.getSensitiveFields({ type: 'key_ring' });
    const result = Array.isArray(data) ? data : data.keys || data.items || [];
    keys.value = result;
    rotationHistory.value = data.history || [];
  } catch (err) {
    keysError.value = err.message || 'Failed to load keys';
  } finally {
    keysLoading.value = false;
  }
}

function daysUntilRotation(row) {
  if (!row.rotates_at) return null;
  const diff = new Date(row.rotates_at) - new Date();
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)));
}

function openRotateConfirm(row) {
  rotatingKey.value = row;
  showRotateConfirm.value = true;
}

async function rotateKey() {
  rotating.value = true;
  try {
    await securityApi.updateSensitiveFields({ action: 'rotate', key_id: rotatingKey.value.key_id });
    toast.value?.addToast({ message: 'Key rotated successfully', type: 'success' });
    showRotateConfirm.value = false;
    await loadKeys();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Rotation failed', type: 'error' });
  } finally {
    rotating.value = false;
  }
}

// ---- Tab switching auto-load ----
watch(activeTab, (tab) => {
  if (tab === 'fields' && fields.value.length === 0) loadFields();
  if (tab === 'retention' && retentionPolicies.value.length === 0) loadRetention();
  if (tab === 'holds') loadHolds();
  if (tab === 'audit') loadAuditRequests();
  if (tab === 'passwords') loadPasswordResets();
  if (tab === 'keys') loadKeys();
});

onMounted(() => { loadFields(); });
</script>

<style lang="scss" scoped>
.tabs {
  display: flex;
  gap: $space-1;
  border-bottom: 1px solid $border-color;
  margin-bottom: $space-6;
  flex-wrap: wrap;
}

.tab-btn {
  padding: $space-3 $space-4;
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  color: $color-neutral-500;
  border-bottom: 2px solid transparent;
  transition: all $transition-fast;
  margin-bottom: -1px;
  white-space: nowrap;
  &:hover { color: $color-neutral-700; }
  &--active { color: $color-primary-600; border-bottom-color: $color-primary-500; }
}

.tab-panel {
  min-height: 300px;
  &__actions {
    display: flex;
    gap: $space-3;
    margin-bottom: $space-5;
    flex-wrap: wrap;
  }
}

.mono-code {
  font-family: $font-family-mono;
  font-size: $font-size-sm;
  background: $color-neutral-50;
  padding: 2px 6px;
  border-radius: $border-radius-sm;
  color: $color-neutral-700;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: $space-1;
}

.masked-preview {
  font-family: $font-family-mono;
  font-size: $font-size-sm;
  color: $color-neutral-500;
  letter-spacing: 1px;
}

// Inline edit
.inline-edit {
  display: flex;
  align-items: center;
  gap: $space-2;
}

.editable-value {
  cursor: pointer;
  border-bottom: 1px dashed $color-neutral-300;
  padding-bottom: 1px;
  transition: color $transition-fast;
  &:hover { color: $color-primary-600; border-bottom-color: $color-primary-400; }
}

// Inline toggle
.inline-toggle {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  input { position: absolute; opacity: 0; width: 0; height: 0; }
  &__track {
    display: block; width: 36px; height: 20px; border-radius: $border-radius-full;
    background: $color-neutral-300; position: relative; transition: background $transition-fast;
    &::after {
      content: ''; position: absolute; top: 2px; left: 2px; width: 16px; height: 16px;
      border-radius: 50%; background: #fff; box-shadow: $shadow-xs; transition: transform $transition-fast;
    }
  }
  input:checked + .inline-toggle__track {
    background: $color-primary-500;
    &::after { transform: translateX(16px); }
  }
}

// Approval indicator
.approval-indicator {
  display: flex;
  align-items: center;
  gap: $space-2;
}

.approval-dots {
  display: flex;
  gap: 4px;
}

.approval-dot {
  width: 12px; height: 12px; border-radius: 50%;
  border: 2px solid $color-neutral-300; background: transparent;
  &--filled { background: $color-success-500; border-color: $color-success-500; }
}

// Token display
.token-display {
  display: flex;
  flex-direction: column;
  gap: $space-1;
  margin-top: $space-2;
}

.token-code {
  padding: $space-2 $space-3;
  background: $color-warning-50;
  border: 1px solid $color-warning-100;
  border-radius: $border-radius-base;
  word-break: break-all;
  font-size: $font-size-sm;
}

// Rotation info
.rotation-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.rotation-history {
  h4 { font-size: $font-size-md; color: $color-neutral-800; }
}

.history-list {
  border: 1px solid $border-color;
  border-radius: $border-radius-md;
  overflow: hidden;
}

.history-entry {
  display: flex;
  align-items: center;
  gap: $space-4;
  padding: $space-3 $space-4;
  border-bottom: 1px solid $color-neutral-50;
  &:last-child { border-bottom: none; }
}

// Purge
.purge-preview {
  .purge-list {
    margin: $space-2 0;
    padding-left: $space-5;
    li { list-style: disc; margin-bottom: $space-1; color: $color-neutral-700; }
  }
}

.purge-warning {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $space-4;
  text-align: center;
  padding: $space-4;
  color: $color-danger-500;
  p { color: $color-neutral-700; max-width: 400px; }
}

// Form
.form-stack {
  display: flex;
  flex-direction: column;
  gap: $space-4;
}
</style>
