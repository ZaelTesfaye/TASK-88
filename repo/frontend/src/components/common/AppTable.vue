<template>
  <div class="app-table-wrap">
    <div class="app-table-container">
      <table class="app-table">
        <thead>
          <tr>
            <th
              v-for="col in columns"
              :key="col.key"
              :class="{
                'app-table__th--sortable': col.sortable,
                'app-table__th--sorted': sortKey === col.key,
              }"
              :style="col.width ? { width: col.width } : {}"
              @click="col.sortable ? toggleSort(col.key) : null"
            >
              <span class="app-table__th-content">
                {{ col.label }}
                <span v-if="col.sortable" class="app-table__sort-icon">
                  <svg v-if="sortKey === col.key && sortOrder === 'asc'" width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M7 3L11 8H3L7 3Z" fill="currentColor" />
                  </svg>
                  <svg v-else-if="sortKey === col.key && sortOrder === 'desc'" width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M7 11L3 6H11L7 11Z" fill="currentColor" />
                  </svg>
                  <svg v-else width="14" height="14" viewBox="0 0 14 14" fill="none" opacity="0.3">
                    <path d="M7 3L10 6.5H4L7 3ZM7 11L4 7.5H10L7 11Z" fill="currentColor" />
                  </svg>
                </span>
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td :colspan="columns.length" class="app-table__loading-cell">
              <div class="app-table__loading">
                <div class="skeleton" style="width: 100%; height: 16px; margin-bottom: 8px" v-for="n in 5" :key="n" />
              </div>
            </td>
          </tr>
          <tr v-else-if="!rows.length">
            <td :colspan="columns.length" class="app-table__empty-cell">
              <slot name="empty">
                <div class="app-table__empty">
                  <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
                    <rect x="4" y="8" width="32" height="24" rx="3" stroke="#CED4DA" stroke-width="1.5" />
                    <path d="M4 14H36" stroke="#CED4DA" stroke-width="1.5" />
                    <path d="M14 14V32" stroke="#CED4DA" stroke-width="1.5" />
                  </svg>
                  <p>No data available</p>
                </div>
              </slot>
            </td>
          </tr>
          <template v-else>
            <tr
              v-for="(row, idx) in rows"
              :key="row.id || idx"
              :class="{ 'app-table__row--clickable': rowClickable }"
              @click="rowClickable ? $emit('row-click', row) : null"
            >
              <td v-for="col in columns" :key="col.key">
                <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
                  {{ row[col.key] ?? '---' }}
                </slot>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <div v-if="totalPages > 1" class="app-table__pagination">
      <span class="app-table__pagination-info">
        Page {{ currentPage }} of {{ totalPages }}
        <span v-if="totalItems" class="text-muted">({{ totalItems }} items)</span>
      </span>
      <div class="app-table__pagination-actions">
        <button
          class="app-table__page-btn"
          :disabled="currentPage <= 1"
          @click="$emit('page-change', currentPage - 1)"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M10 4L6 8L10 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
        <button
          v-for="page in visiblePages"
          :key="page"
          class="app-table__page-btn"
          :class="{ 'app-table__page-btn--active': page === currentPage }"
          @click="typeof page === 'number' ? $emit('page-change', page) : null"
          :disabled="typeof page !== 'number'"
        >
          {{ page }}
        </button>
        <button
          class="app-table__page-btn"
          :disabled="currentPage >= totalPages"
          @click="$emit('page-change', currentPage + 1)"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M6 4L10 8L6 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  columns: {
    type: Array,
    required: true,
  },
  rows: {
    type: Array,
    default: () => [],
  },
  loading: Boolean,
  sortKey: String,
  sortOrder: {
    type: String,
    default: 'asc',
  },
  currentPage: {
    type: Number,
    default: 1,
  },
  totalPages: {
    type: Number,
    default: 1,
  },
  totalItems: {
    type: Number,
    default: 0,
  },
  rowClickable: Boolean,
});

const emit = defineEmits(['sort', 'page-change', 'row-click']);

function toggleSort(key) {
  if (props.sortKey === key) {
    emit('sort', { key, order: props.sortOrder === 'asc' ? 'desc' : 'asc' });
  } else {
    emit('sort', { key, order: 'asc' });
  }
}

const visiblePages = computed(() => {
  const total = props.totalPages;
  const current = props.currentPage;
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const pages = [];
  pages.push(1);
  if (current > 3) pages.push('...');
  for (let i = Math.max(2, current - 1); i <= Math.min(total - 1, current + 1); i++) {
    pages.push(i);
  }
  if (current < total - 2) pages.push('...');
  pages.push(total);
  return pages;
});
</script>

<style lang="scss" scoped>
.app-table-wrap {
  width: 100%;
}

.app-table-container {
  overflow-x: auto;
  border: 1px solid $border-color;
  border-radius: $border-radius-md;
  background: $color-neutral-0;
}

.app-table {
  width: 100%;
  border-collapse: collapse;

  th, td {
    padding: $space-3 $space-4;
    text-align: left;
    font-size: $font-size-base;
  }

  thead {
    background: $color-neutral-25;
    border-bottom: 1px solid $border-color;
  }

  th {
    font-weight: $font-weight-semibold;
    color: $color-neutral-600;
    font-size: $font-size-sm;
    text-transform: uppercase;
    letter-spacing: 0.3px;
    white-space: nowrap;
  }

  tbody tr {
    border-bottom: 1px solid $color-neutral-50;
    transition: background $transition-fast;

    &:last-child {
      border-bottom: none;
    }

    &:hover {
      background: $color-neutral-25;
    }
  }

  td {
    color: $color-neutral-700;
  }

  &__th--sortable {
    cursor: pointer;
    user-select: none;

    &:hover {
      color: $color-neutral-800;
      background: $color-neutral-50;
    }
  }

  &__th--sorted {
    color: $color-primary-600;
  }

  &__th-content {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  &__sort-icon {
    display: inline-flex;
    color: $color-neutral-400;
  }

  &__loading-cell,
  &__empty-cell {
    padding: $space-8 $space-4;
  }

  &__loading {
    max-width: 600px;
  }

  &__empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: $space-3;
    color: $color-neutral-400;

    p {
      font-size: $font-size-base;
    }
  }

  &__row--clickable {
    cursor: pointer;
  }

  &__pagination {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: $space-3 0;
    margin-top: $space-3;
  }

  &__pagination-info {
    font-size: $font-size-sm;
    color: $color-neutral-500;
  }

  &__pagination-actions {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  &__page-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 32px;
    height: 32px;
    padding: 0 8px;
    border: 1px solid transparent;
    border-radius: $border-radius-base;
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    color: $color-neutral-600;
    cursor: pointer;
    transition: all $transition-fast;

    &:hover:not(:disabled) {
      background: $color-neutral-50;
      border-color: $border-color;
    }

    &:disabled {
      opacity: 0.4;
      cursor: not-allowed;
    }

    &--active {
      background: $color-primary-500;
      color: #fff;
      border-color: $color-primary-500;

      &:hover:not(:disabled) {
        background: $color-primary-600;
        border-color: $color-primary-600;
      }
    }
  }
}
</style>
