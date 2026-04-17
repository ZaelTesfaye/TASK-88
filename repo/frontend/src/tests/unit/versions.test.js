import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGet = vi.fn().mockResolvedValue({ data: {} });
const mockPost = vi.fn().mockResolvedValue({ data: {} });
const mockDelete = vi.fn().mockResolvedValue({ data: {} });

vi.mock('@/api/client.js', () => ({
  default: {
    get: (...args) => mockGet(...args),
    post: (...args) => mockPost(...args),
    delete: (...args) => mockDelete(...args),
  },
  setLogoutCallback: vi.fn(),
}));

import {
  listVersions,
  getVersion,
  getVersionItems,
  diffVersions,
  createVersion,
  submitReview,
  addVersionItem,
  removeVersionItem,
  activate,
} from '@/api/versions.js';

describe('Versions API adapter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('listVersions calls GET /versions/:entity', async () => {
    await listVersions('sku', { page: 1 });
    expect(mockGet).toHaveBeenCalledWith('/versions/sku', { params: { page: 1 } });
  });

  it('getVersion calls GET /versions/:entity/:id', async () => {
    await getVersion('sku', 42);
    expect(mockGet).toHaveBeenCalledWith('/versions/sku/42');
  });

  it('getVersionItems calls GET /versions/:entity/:id/items', async () => {
    await getVersionItems('sku', 42, { page: 2 });
    expect(mockGet).toHaveBeenCalledWith('/versions/sku/42/items', { params: { page: 2 } });
  });

  it('diffVersions calls GET /versions/:entity/:id/diff', async () => {
    await diffVersions('sku', 42);
    expect(mockGet).toHaveBeenCalledWith('/versions/sku/42/diff', { params: {} });
  });

  it('createVersion calls POST /versions/:entity', async () => {
    const payload = { scope_key: 'node:1' };
    await createVersion('sku', payload);
    expect(mockPost).toHaveBeenCalledWith('/versions/sku', payload);
  });

  it('submitReview calls POST /versions/:entity/:id/review', async () => {
    await submitReview('sku', 42);
    expect(mockPost).toHaveBeenCalledWith('/versions/sku/42/review');
  });

  it('addVersionItem calls POST /versions/:entity/:id/items', async () => {
    const payload = { master_record_ids: [1, 2] };
    await addVersionItem('sku', 42, payload);
    expect(mockPost).toHaveBeenCalledWith('/versions/sku/42/items', payload);
  });

  it('removeVersionItem calls DELETE /versions/:entity/:id/items/:itemId', async () => {
    await removeVersionItem('sku', 42, 7);
    expect(mockDelete).toHaveBeenCalledWith('/versions/sku/42/items/7');
  });

  it('activate calls POST /versions/:entity/:id/activate', async () => {
    await activate('sku', 42);
    expect(mockPost).toHaveBeenCalledWith('/versions/sku/42/activate');
  });
});
