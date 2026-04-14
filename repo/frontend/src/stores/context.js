import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import * as orgApi from '@/api/org.js';

const CONTEXT_CHANGED_EVENT = 'context:changed';

export const useContextStore = defineStore('context', () => {
  // ---- State ----
  const currentNode = ref(null);
  const breadcrumb = ref([]);
  const descendantNodeIds = ref([]);
  const loading = ref(false);

  // ---- Getters ----
  const scopeFilter = computed(() => {
    if (!currentNode.value) return {};
    return {
      nodeId: currentNode.value.id,
      descendantNodeIds: descendantNodeIds.value,
    };
  });

  const contextBreadcrumb = computed(() =>
    breadcrumb.value.map((node) => ({
      id: node.id,
      label: node.name,
      level: node.level,
    }))
  );

  // ---- Actions ----
  async function switchContext(nodeId) {
    loading.value = true;
    try {
      const { data: resp } = await orgApi.switchContext(nodeId);
      currentNode.value = resp.data.current_node;
      breadcrumb.value = resp.data.breadcrumb || [];
      await loadDescendants();
      emitContextChanged();
    } finally {
      loading.value = false;
    }
  }

  async function loadDescendants() {
    if (!currentNode.value) {
      descendantNodeIds.value = [];
      return;
    }
    try {
      const { data: resp } = await orgApi.getOrgTree();
      descendantNodeIds.value = collectDescendantIds(resp.data, currentNode.value.id);
    } catch {
      descendantNodeIds.value = [];
    }
  }

  function clearContext() {
    currentNode.value = null;
    breadcrumb.value = [];
    descendantNodeIds.value = [];
    emitContextChanged();
  }

  function emitContextChanged() {
    window.dispatchEvent(new CustomEvent(CONTEXT_CHANGED_EVENT, {
      detail: { nodeId: currentNode.value?.id || null },
    }));
  }

  return {
    currentNode,
    breadcrumb,
    descendantNodeIds,
    loading,
    scopeFilter,
    contextBreadcrumb,
    switchContext,
    loadDescendants,
    clearContext,
  };
});

/**
 * Recursively collect all descendant node IDs for a given nodeId from a tree.
 */
function collectDescendantIds(tree, targetId) {
  const ids = [];
  function findAndCollect(nodes, collecting) {
    for (const node of nodes) {
      if (node.id === targetId || collecting) {
        if (node.id !== targetId) {
          ids.push(node.id);
        }
        if (node.children) {
          findAndCollect(node.children, true);
        }
      } else if (node.children) {
        findAndCollect(node.children, false);
      }
    }
  }
  const root = Array.isArray(tree) ? tree : [tree];
  findAndCollect(root, false);
  return ids;
}

export { CONTEXT_CHANGED_EVENT };
