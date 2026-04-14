<template>
  <button
    :class="[
      'app-btn',
      `app-btn--${variant}`,
      `app-btn--${size}`,
      {
        'app-btn--loading': loading,
        'app-btn--icon-only': iconOnly,
        'app-btn--block': block,
      },
    ]"
    :disabled="disabled || loading"
    :type="type"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="app-btn__spinner" aria-hidden="true">
      <svg viewBox="0 0 20 20" fill="none">
        <circle cx="10" cy="10" r="8" stroke="currentColor" stroke-width="2" opacity="0.3" />
        <path
          d="M10 2a8 8 0 0 1 8 8"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
        />
      </svg>
    </span>
    <span class="app-btn__content" :class="{ 'app-btn__content--hidden': loading }">
      <slot />
    </span>
  </button>
</template>

<script setup>
defineProps({
  variant: {
    type: String,
    default: 'primary',
    validator: (v) => ['primary', 'secondary', 'outline', 'ghost', 'danger', 'success'].includes(v),
  },
  size: {
    type: String,
    default: 'md',
    validator: (v) => ['sm', 'md', 'lg'].includes(v),
  },
  type: {
    type: String,
    default: 'button',
  },
  loading: Boolean,
  disabled: Boolean,
  iconOnly: Boolean,
  block: Boolean,
});

defineEmits(['click']);
</script>

<style lang="scss" scoped>
.app-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid transparent;
  border-radius: $border-radius-base;
  font-weight: $font-weight-medium;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
  transition: all $transition-fast;
  outline: none;

  &:focus-visible {
    box-shadow: 0 0 0 2px $color-primary-200;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
  }

  // Sizes
  &--sm {
    height: 32px;
    padding: 0 12px;
    font-size: $font-size-sm;
  }

  &--md {
    height: 36px;
    padding: 0 16px;
    font-size: $font-size-base;
  }

  &--lg {
    height: 44px;
    padding: 0 24px;
    font-size: $font-size-md;
  }

  // Variants
  &--primary {
    background: $color-primary-500;
    color: #fff;

    &:hover:not(:disabled) {
      background: $color-primary-600;
    }

    &:active:not(:disabled) {
      background: $color-primary-700;
    }
  }

  &--secondary {
    background: $color-neutral-100;
    color: $color-neutral-700;

    &:hover:not(:disabled) {
      background: $color-neutral-200;
      color: $color-neutral-800;
    }

    &:active:not(:disabled) {
      background: $color-neutral-300;
    }
  }

  &--outline {
    border-color: $border-color;
    background: transparent;
    color: $color-neutral-700;

    &:hover:not(:disabled) {
      border-color: $color-neutral-300;
      background: $color-neutral-50;
    }

    &:active:not(:disabled) {
      background: $color-neutral-100;
    }
  }

  &--ghost {
    background: transparent;
    color: $color-neutral-600;

    &:hover:not(:disabled) {
      background: $color-neutral-50;
      color: $color-neutral-800;
    }

    &:active:not(:disabled) {
      background: $color-neutral-100;
    }
  }

  &--danger {
    background: $color-danger-500;
    color: #fff;

    &:hover:not(:disabled) {
      background: $color-danger-600;
    }

    &:active:not(:disabled) {
      background: $color-danger-700;
    }

    &:focus-visible {
      box-shadow: 0 0 0 2px $color-danger-100;
    }
  }

  &--success {
    background: $color-success-500;
    color: #fff;

    &:hover:not(:disabled) {
      background: $color-success-600;
    }

    &:active:not(:disabled) {
      background: $color-success-700;
    }

    &:focus-visible {
      box-shadow: 0 0 0 2px $color-success-100;
    }
  }

  // States
  &--loading {
    cursor: wait;
  }

  &--icon-only {
    &.app-btn--sm { width: 32px; padding: 0; }
    &.app-btn--md { width: 36px; padding: 0; }
    &.app-btn--lg { width: 44px; padding: 0; }
  }

  &--block {
    width: 100%;
  }

  // Spinner
  &__spinner {
    position: absolute;
    display: flex;
    align-items: center;
    justify-content: center;

    svg {
      width: 18px;
      height: 18px;
      animation: spin 0.8s linear infinite;
    }
  }

  &__content {
    display: inline-flex;
    align-items: center;
    gap: 6px;

    &--hidden {
      visibility: hidden;
    }
  }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
