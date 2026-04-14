<template>
  <span
    :class="['app-chip', `app-chip--${resolvedVariant}`, { 'app-chip--sm': size === 'sm' }]"
  >
    <span class="app-chip__dot" />
    <span class="app-chip__label">{{ label }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  status: {
    type: String,
    default: '',
  },
  label: {
    type: String,
    default: '',
  },
  variant: {
    type: String,
    default: '',
  },
  size: {
    type: String,
    default: 'md',
  },
});

const STATUS_VARIANT_MAP = {
  active: 'success',
  effective: 'success',
  ready: 'success',
  inactive: 'neutral',
  archived: 'neutral',
  draft: 'info',
  review: 'warning',
  pending: 'warning',
  running: 'info',
  blocked: 'warning',
  'awaiting-ack': 'warning',
  failed: 'danger',
  error: 'danger',
};

const resolvedVariant = computed(() => {
  if (props.variant) return props.variant;
  return STATUS_VARIANT_MAP[props.status] || 'neutral';
});
</script>

<style lang="scss" scoped>
.app-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: $border-radius-full;
  font-size: $font-size-xs;
  font-weight: $font-weight-medium;
  white-space: nowrap;
  line-height: 1.4;

  &--sm {
    padding: 2px 8px;
    font-size: 10px;
  }

  &__dot {
    width: 6px;
    height: 6px;
    border-radius: $border-radius-full;
    flex-shrink: 0;
  }

  &--success {
    background: $color-success-50;
    color: $color-success-700;
    .app-chip__dot { background: $color-success-500; }
  }

  &--warning {
    background: $color-warning-50;
    color: $color-warning-700;
    .app-chip__dot { background: $color-warning-500; }
  }

  &--danger {
    background: $color-danger-50;
    color: $color-danger-700;
    .app-chip__dot { background: $color-danger-500; }
  }

  &--info {
    background: $color-primary-50;
    color: $color-primary-700;
    .app-chip__dot { background: $color-primary-500; }
  }

  &--neutral {
    background: $color-neutral-100;
    color: $color-neutral-600;
    .app-chip__dot { background: $color-neutral-400; }
  }
}
</style>
