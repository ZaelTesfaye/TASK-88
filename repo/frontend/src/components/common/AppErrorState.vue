<template>
  <div class="app-error">
    <div class="app-error__icon">
      <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
        <circle cx="24" cy="24" r="18" stroke="currentColor" stroke-width="1.5"/>
        <path d="M24 14V26" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        <circle cx="24" cy="32" r="1.5" fill="currentColor"/>
      </svg>
    </div>
    <h4 class="app-error__title">{{ title }}</h4>
    <p v-if="message" class="app-error__message">{{ message }}</p>
    <p v-if="code" class="app-error__code">Error code: {{ code }}</p>
    <div v-if="retryable" class="app-error__action">
      <button class="app-error__retry-btn" @click="$emit('retry')">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <path d="M2 8C2 4.686 4.686 2 8 2C10.21 2 12.117 3.217 13.167 5M14 8C14 11.314 11.314 14 8 14C5.79 14 3.883 12.783 2.833 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          <path d="M10 5H14V1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        Try Again
      </button>
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    default: 'Something went wrong',
  },
  message: String,
  code: String,
  retryable: {
    type: Boolean,
    default: true,
  },
});

defineEmits(['retry']);
</script>

<style lang="scss" scoped>
.app-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: $space-10 $space-6;
  text-align: center;

  &__icon {
    color: $color-danger-400;
    margin-bottom: $space-4;
  }

  &__title {
    font-size: $font-size-md;
    font-weight: $font-weight-semibold;
    color: $color-neutral-700;
    margin: 0 0 $space-2;
  }

  &__message {
    font-size: $font-size-base;
    color: $color-neutral-500;
    max-width: 400px;
    margin: 0 0 $space-1;
  }

  &__code {
    font-size: $font-size-xs;
    color: $color-neutral-400;
    font-family: $font-family-mono;
    margin: 0;
  }

  &__action {
    margin-top: $space-5;
  }

  &__retry-btn {
    display: inline-flex;
    align-items: center;
    gap: $space-2;
    padding: $space-2 $space-4;
    border: 1px solid $border-color;
    border-radius: $border-radius-base;
    font-size: $font-size-base;
    font-weight: $font-weight-medium;
    color: $color-primary-500;
    background: $color-neutral-0;
    cursor: pointer;
    transition: all $transition-fast;

    &:hover {
      border-color: $color-primary-300;
      background: $color-primary-50;
    }
  }
}
</style>
