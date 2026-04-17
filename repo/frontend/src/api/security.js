import client from "./client.js";

export function getSensitiveFields(params = {}) {
  return client.get("/security/sensitive-fields", { params });
}

export function createSensitiveField(payload) {
  return client.post("/security/sensitive-fields", payload);
}

export function updateSensitiveField(fieldId, payload) {
  return client.put(`/security/sensitive-fields/${fieldId}`, payload);
}

export function deleteSensitiveField(fieldId) {
  return client.delete(`/security/sensitive-fields/${fieldId}`);
}

export function getKeys() {
  return client.get("/security/keys");
}

export function getKey(keyId) {
  return client.get(`/security/keys/${keyId}`);
}

export function rotateKey(payload) {
  return client.post("/security/keys/rotate", payload);
}

export function createPasswordResetRequest(payload) {
  return client.post("/security/password-reset", payload);
}

export function approvePasswordResetRequest(requestId) {
  return client.post(`/security/password-reset/${requestId}/approve`);
}

export function getPasswordResetRequests(params = {}) {
  return client.get("/security/password-reset", { params });
}

export function getRetentionPolicies(params = {}) {
  return client.get("/security/retention-policies", { params });
}

export function createRetentionPolicy(payload) {
  return client.post("/security/retention-policies", payload);
}

export function updateRetentionPolicy(policyId, payload) {
  return client.put(`/security/retention-policies/${policyId}`, payload);
}

export function getLegalHolds(params = {}) {
  return client.get("/security/legal-holds", { params });
}

export function createLegalHold(payload) {
  return client.post("/security/legal-holds", payload);
}

export function releaseLegalHold(holdId) {
  return client.post(`/security/legal-holds/${holdId}/release`);
}

export function dryRunPurge(params = {}) {
  return client.post("/security/purge-runs/dry-run", params);
}

export function executePurge(params = {}) {
  return client.post("/security/purge-runs/execute", params);
}

export function getPurgeRuns(params = {}) {
  return client.get("/security/purge-runs", { params });
}

// Compatibility aliases used by page components.
export function updateRetentionPolicies(policyId, payload) {
  return updateRetentionPolicy(policyId, payload);
}

export async function updateSensitiveFields(payload) {
  const fieldId = payload?.id ?? payload?.field_id;
  if (fieldId) {
    return updateSensitiveField(fieldId, payload);
  }

  if (payload?.field_key) {
    const { data } = await getSensitiveFields();
    const items = Array.isArray(data) ? data : data?.items || [];
    const matched = items.find((item) => item.field_key === payload.field_key);
    if (matched?.id) {
      return updateSensitiveField(matched.id, payload);
    }
  }

  throw new Error("Unable to resolve sensitive field ID for update");
}
