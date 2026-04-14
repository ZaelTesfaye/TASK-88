import client from './client.js';

export function getOrgTree() {
  return client.get('/org/tree');
}

export function createNode(parentId, payload) {
  return client.post(`/org/nodes`, { ...payload, parent_id: parentId });
}

export function updateNode(nodeId, payload) {
  return client.put(`/org/nodes/${nodeId}`, payload);
}

export function deleteNode(nodeId) {
  return client.delete(`/org/nodes/${nodeId}`);
}

export function switchContext(nodeId) {
  return client.post('/context/switch', { node_id: nodeId });
}

export function getCurrentContext() {
  return client.get('/context/current');
}
