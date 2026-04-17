import client from "./client.js";

export function getSources(params = {}) {
  return client.get("/ingestion/sources", { params });
}

export function createSource(payload) {
  return client.post("/ingestion/sources", payload);
}

export function getSource(sourceId) {
  return client.get(`/ingestion/sources/${sourceId}`);
}

export function updateSource(sourceId, payload) {
  return client.put(`/ingestion/sources/${sourceId}`, payload);
}

export function deleteSource(sourceId) {
  return client.delete(`/ingestion/sources/${sourceId}`);
}

export function getJobs(params = {}) {
  return client.get("/ingestion/jobs", { params });
}

export function createJob(payload) {
  return client.post("/ingestion/jobs", payload);
}

export function getJob(jobId) {
  return client.get(`/ingestion/jobs/${jobId}`);
}

export function retryJob(jobId) {
  return client.post(`/ingestion/jobs/${jobId}/retry`);
}

export function acknowledgeJob(jobId, payload = {}) {
  return client.post(`/ingestion/jobs/${jobId}/acknowledge`, payload);
}

export function getCheckpoints(jobId) {
  return client.get(`/ingestion/jobs/${jobId}/checkpoints`);
}

export function getFailures(jobId, params = {}) {
  return client.get(`/ingestion/jobs/${jobId}/failures`, { params });
}

// Compatibility aliases used by page components.
export function getConnectorHealth(connectorId) {
  return client.get(`/ingestion/connectors/${connectorId}/health`);
}

export function getCapabilities(connectorId) {
  return client.get(`/ingestion/connectors/${connectorId}/capabilities`);
}

export function runJob(importSourceId, options = {}) {
  return client.post("/ingestion/jobs/run", {
    import_source_id: importSourceId,
    priority: options.priority ?? 0,
    dependency_group: options.dependency_group ?? "",
    mode: options.mode ?? "incremental",
  });
}
