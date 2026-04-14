import client from './client.js';

export function getEndpoints(params = {}) {
  return client.get('/integrations/endpoints', { params });
}

export function createEndpoint(payload) {
  return client.post('/integrations/endpoints', payload);
}

export function getEndpoint(endpointId) {
  return client.get(`/integrations/endpoints/${endpointId}`);
}

export function updateEndpoint(endpointId, payload) {
  return client.put(`/integrations/endpoints/${endpointId}`, payload);
}

export function deleteEndpoint(endpointId) {
  return client.delete(`/integrations/endpoints/${endpointId}`);
}

export function testEndpoint(endpointId) {
  return client.post(`/integrations/endpoints/${endpointId}/test`);
}

export function getDeliveries(params = {}) {
  return client.get('/integrations/deliveries', { params });
}

export function getDelivery(deliveryId) {
  return client.get(`/integrations/deliveries/${deliveryId}`);
}

export function retryDelivery(deliveryId) {
  return client.post(`/integrations/deliveries/${deliveryId}/retry`);
}

export function getConnectors(params = {}) {
  return client.get('/integrations/connectors', { params });
}

export function createConnector(payload) {
  return client.post('/integrations/connectors', payload);
}

export function getConnector(connectorId) {
  return client.get(`/integrations/connectors/${connectorId}`);
}

export function updateConnector(connectorId, payload) {
  return client.put(`/integrations/connectors/${connectorId}`, payload);
}

export function deleteConnector(connectorId) {
  return client.delete(`/integrations/connectors/${connectorId}`);
}

export function healthCheckConnector(connectorId) {
  return client.post(`/integrations/connectors/${connectorId}/health-check`);
}
