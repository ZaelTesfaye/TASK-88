<template>
  <nav class="app-breadcrumb" aria-label="Context breadcrumb">
    <ol class="app-breadcrumb__list">
      <li
        v-for="(item, index) in items"
        :key="item.id || index"
        class="app-breadcrumb__item"
      >
        <span v-if="index > 0" class="app-breadcrumb__separator">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M5 3L9 7L5 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </span>
        <button
          v-if="index < items.length - 1"
          class="app-breadcrumb__link"
          @click="$emit('navigate', item)"
        >
          {{ item.label }}
        </button>
        <span v-else class="app-breadcrumb__current">{{ item.label }}</span>
      </li>
    </ol>
  </nav>
</template>

<script setup>
defineProps({
  items: {
    type: Array,
    default: () => [],
  },
});

defineEmits(['navigate']);
</script>

<style lang="scss" scoped>
.app-breadcrumb {
  &__list {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 2px;
  }

  &__item {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  &__separator {
    display: flex;
    color: $color-neutral-300;
  }

  &__link {
    font-size: $font-size-sm;
    color: $color-primary-500;
    font-weight: $font-weight-medium;
    padding: 2px 4px;
    border-radius: $border-radius-sm;
    transition: all $transition-fast;

    &:hover {
      background: $color-primary-50;
      text-decoration: none;
    }
  }

  &__current {
    font-size: $font-size-sm;
    color: $color-neutral-600;
    font-weight: $font-weight-medium;
    padding: 2px 4px;
  }
}
</style>
