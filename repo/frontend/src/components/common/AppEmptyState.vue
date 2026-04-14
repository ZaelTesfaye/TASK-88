<template>
  <div class="app-empty">
    <div class="app-empty__icon">
      <slot name="icon">
        <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
          <rect x="6" y="10" width="36" height="28" rx="4" stroke="currentColor" stroke-width="1.5"/>
          <path d="M6 18H42" stroke="currentColor" stroke-width="1.5"/>
          <path d="M18 18V38" stroke="currentColor" stroke-width="1.5"/>
          <circle cx="12" cy="14" r="1.5" fill="currentColor" opacity="0.4"/>
          <circle cx="16.5" cy="14" r="1.5" fill="currentColor" opacity="0.4"/>
        </svg>
      </slot>
    </div>
    <h4 class="app-empty__title">{{ title }}</h4>
    <p v-if="description" class="app-empty__description">{{ description }}</p>
    <div v-if="$slots.action" class="app-empty__action">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    default: 'No data found',
  },
  description: String,
});
</script>

<style lang="scss" scoped>
.app-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: $space-10 $space-6;
  text-align: center;

  &__icon {
    color: $color-neutral-300;
    margin-bottom: $space-4;
  }

  &__title {
    font-size: $font-size-md;
    font-weight: $font-weight-semibold;
    color: $color-neutral-600;
    margin: 0 0 $space-2;
  }

  &__description {
    font-size: $font-size-base;
    color: $color-neutral-400;
    max-width: 360px;
    margin: 0;
  }

  &__action {
    margin-top: $space-5;
  }
}
</style>
