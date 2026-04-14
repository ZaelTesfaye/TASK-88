import { describe, it, expect, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';

// Mock the org API
vi.mock('@/api/org.js', () => ({
  switchContext: vi.fn(),
  getOrgTree: vi.fn(),
  getCurrentContext: vi.fn(),
}));

// Mock the api/client.js module
vi.mock('@/api/client.js', () => ({
  default: { post: vi.fn(), get: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

import { useContextStore } from '@/stores/context.js';
import * as orgApi from '@/api/org.js';

// Helper: builds the mock response matching the real backend shape
// Backend returns: { data: { current_node, scope_ids, breadcrumb } }
// Axios wraps that in another { data: ... }
function mockSwitchResponse(node, breadcrumb = []) {
  return {
    data: {
      data: {
        current_node: node,
        scope_ids: [],
        breadcrumb,
      },
    },
  };
}

// Helper for getOrgTree — backend returns { data: tree }, axios wraps in { data: ... }
function mockTreeResponse(tree) {
  return {
    data: {
      data: tree,
    },
  };
}

describe('Context Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('switchContext updates currentNode', async () => {
    const mockNode = { id: 'node-1', name: 'Region A', level: 'region' };
    orgApi.switchContext.mockResolvedValue(mockSwitchResponse(mockNode, [
      { id: 'root', name: 'Global', level: 'root' },
      mockNode,
    ]));
    orgApi.getOrgTree.mockResolvedValue(mockTreeResponse([
      {
        id: 'root',
        name: 'Global',
        children: [
          {
            id: 'node-1',
            name: 'Region A',
            children: [
              { id: 'node-1a', name: 'Sub A1', children: [] },
              { id: 'node-1b', name: 'Sub A2', children: [] },
            ],
          },
        ],
      },
    ]));

    const store = useContextStore();
    expect(store.currentNode).toBeNull();

    await store.switchContext('node-1');

    expect(store.currentNode).toEqual(mockNode);
    expect(orgApi.switchContext).toHaveBeenCalledWith('node-1');
  });

  it('switchContext updates breadcrumb', async () => {
    const breadcrumb = [
      { id: 'root', name: 'Global', level: 'root' },
      { id: 'node-1', name: 'Region A', level: 'region' },
    ];
    orgApi.switchContext.mockResolvedValue(mockSwitchResponse(
      { id: 'node-1', name: 'Region A', level: 'region' },
      breadcrumb,
    ));
    orgApi.getOrgTree.mockResolvedValue(mockTreeResponse([]));

    const store = useContextStore();
    await store.switchContext('node-1');

    expect(store.breadcrumb).toEqual(breadcrumb);
    expect(store.contextBreadcrumb).toEqual([
      { id: 'root', label: 'Global', level: 'root' },
      { id: 'node-1', label: 'Region A', level: 'region' },
    ]);
  });

  it('scopeFilter includes descendant nodes', async () => {
    orgApi.switchContext.mockResolvedValue(mockSwitchResponse(
      { id: 'node-1', name: 'Region A' },
      [{ id: 'node-1', name: 'Region A' }],
    ));
    orgApi.getOrgTree.mockResolvedValue(mockTreeResponse([
      {
        id: 'root',
        name: 'Global',
        children: [
          {
            id: 'node-1',
            name: 'Region A',
            children: [
              { id: 'child-1', name: 'Child 1', children: [] },
              {
                id: 'child-2',
                name: 'Child 2',
                children: [{ id: 'grandchild-1', name: 'Grandchild 1', children: [] }],
              },
            ],
          },
        ],
      },
    ]));

    const store = useContextStore();
    await store.switchContext('node-1');

    expect(store.scopeFilter.nodeId).toBe('node-1');
    expect(store.scopeFilter.descendantNodeIds).toContain('child-1');
    expect(store.scopeFilter.descendantNodeIds).toContain('child-2');
    expect(store.scopeFilter.descendantNodeIds).toContain('grandchild-1');
    expect(store.scopeFilter.descendantNodeIds).toHaveLength(3);
  });

  it('clearContext resets all context state', async () => {
    orgApi.switchContext.mockResolvedValue(mockSwitchResponse(
      { id: 'node-1', name: 'Region A' },
      [{ id: 'node-1', name: 'Region A' }],
    ));
    orgApi.getOrgTree.mockResolvedValue(mockTreeResponse([
      {
        id: 'node-1',
        name: 'Region A',
        children: [{ id: 'child-1', name: 'Child 1', children: [] }],
      },
    ]));

    const store = useContextStore();
    await store.switchContext('node-1');

    expect(store.currentNode).not.toBeNull();
    expect(store.breadcrumb.length).toBeGreaterThan(0);

    store.clearContext();

    expect(store.currentNode).toBeNull();
    expect(store.breadcrumb).toEqual([]);
    expect(store.descendantNodeIds).toEqual([]);
    expect(store.scopeFilter).toEqual({});
  });

  it('scopeFilter returns empty object when no node selected', () => {
    const store = useContextStore();
    expect(store.scopeFilter).toEqual({});
  });

  it('loading state is set correctly during switchContext', async () => {
    let resolveSwitch;
    orgApi.switchContext.mockImplementation(
      () => new Promise((resolve) => { resolveSwitch = resolve; })
    );
    orgApi.getOrgTree.mockResolvedValue(mockTreeResponse([]));

    const store = useContextStore();
    expect(store.loading).toBe(false);

    const switchPromise = store.switchContext('node-1');
    expect(store.loading).toBe(true);

    resolveSwitch(mockSwitchResponse({ id: 'node-1', name: 'Test' }, []));
    await switchPromise;

    expect(store.loading).toBe(false);
  });
});
