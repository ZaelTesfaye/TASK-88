import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGet = vi.fn().mockResolvedValue({ data: {} });
const mockPost = vi.fn().mockResolvedValue({ data: {} });
const mockPut = vi.fn().mockResolvedValue({ data: {} });
const mockDelete = vi.fn().mockResolvedValue({ data: {} });

vi.mock('@/api/client.js', () => ({
  default: {
    get: (...args) => mockGet(...args),
    post: (...args) => mockPost(...args),
    put: (...args) => mockPut(...args),
    delete: (...args) => mockDelete(...args),
  },
  setLogoutCallback: vi.fn(),
}));

import {
  getEndpoints,
  createEndpoint,
  getEndpoint,
  updateEndpoint,
  deleteEndpoint,
  testEndpoint,
  getDeliveries,
  getDelivery,
  retryDelivery,
  getConnectors,
  createConnector,
  getConnector,
  updateConnector,
  deleteConnector,
  healthCheckConnector,
} from '@/api/integrations.js';

describe('Integrations API adapter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ---- Endpoints ----

  it('getEndpoints calls GET /integrations/endpoints', async () => {
    await getEndpoints({ page: 1 });
    expect(mockGet).toHaveBeenCalledWith('/integrations/endpoints', { params: { page: 1 } });
  });

  it('createEndpoint calls POST /integrations/endpoints', async () => {
    const payload = { name: 'Hook', url: 'https://test.com' };
    await createEndpoint(payload);
    expect(mockPost).toHaveBeenCalledWith('/integrations/endpoints', payload);
  });

  it('getEndpoint calls GET /integrations/endpoints/:id', async () => {
    await getEndpoint(5);
    expect(mockGet).toHaveBeenCalledWith('/integrations/endpoints/5');
  });

  it('updateEndpoint calls PUT /integrations/endpoints/:id', async () => {
    const payload = { name: 'Updated' };
    await updateEndpoint(5, payload);
    expect(mockPut).toHaveBeenCalledWith('/integrations/endpoints/5', payload);
  });

  it('deleteEndpoint calls DELETE /integrations/endpoints/:id', async () => {
    await deleteEndpoint(5);
    expect(mockDelete).toHaveBeenCalledWith('/integrations/endpoints/5');
  });

  it('testEndpoint calls POST /integrations/endpoints/:id/test', async () => {
    await testEndpoint(5);
    expect(mockPost).toHaveBeenCalledWith('/integrations/endpoints/5/test');
  });

  // ---- Deliveries ----

  it('getDeliveries calls GET /integrations/deliveries', async () => {
    await getDeliveries({ endpoint_id: 3 });
    expect(mockGet).toHaveBeenCalledWith('/integrations/deliveries', { params: { endpoint_id: 3 } });
  });

  it('getDelivery calls GET /integrations/deliveries/:id', async () => {
    await getDelivery(10);
    expect(mockGet).toHaveBeenCalledWith('/integrations/deliveries/10');
  });

  it('retryDelivery calls POST /integrations/deliveries/:id/retry', async () => {
    await retryDelivery(10);
    expect(mockPost).toHaveBeenCalledWith('/integrations/deliveries/10/retry');
  });

  // ---- Connectors ----

  it('getConnectors calls GET /integrations/connectors', async () => {
    await getConnectors();
    expect(mockGet).toHaveBeenCalledWith('/integrations/connectors', { params: {} });
  });

  it('createConnector calls POST /integrations/connectors', async () => {
    const payload = { name: 'DB Conn', connector_type: 'database' };
    await createConnector(payload);
    expect(mockPost).toHaveBeenCalledWith('/integrations/connectors', payload);
  });

  it('getConnector calls GET /integrations/connectors/:id', async () => {
    await getConnector(8);
    expect(mockGet).toHaveBeenCalledWith('/integrations/connectors/8');
  });

  it('updateConnector calls PUT /integrations/connectors/:id', async () => {
    const payload = { name: 'Updated Conn' };
    await updateConnector(8, payload);
    expect(mockPut).toHaveBeenCalledWith('/integrations/connectors/8', payload);
  });

  it('deleteConnector calls DELETE /integrations/connectors/:id', async () => {
    await deleteConnector(8);
    expect(mockDelete).toHaveBeenCalledWith('/integrations/connectors/8');
  });

  it('healthCheckConnector calls POST /integrations/connectors/:id/health-check', async () => {
    await healthCheckConnector(8);
    expect(mockPost).toHaveBeenCalledWith('/integrations/connectors/8/health-check');
  });
});
