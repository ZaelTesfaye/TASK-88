<template>
  <Teleport to="body">
    <TransitionGroup name="toast" tag="div" class="toast-container">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="['toast', `toast--${toast.type}`]"
        role="alert"
      >
        <span class="toast__icon">
          <svg v-if="toast.type === 'success'" width="18" height="18" viewBox="0 0 18 18" fill="none">
            <path d="M15 4.5L6.75 12.75L3 9" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <svg v-else-if="toast.type === 'error'" width="18" height="18" viewBox="0 0 18 18" fill="none">
            <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="1.5"/>
            <path d="M9 5.5V9.5M9 12V12.01" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <svg v-else-if="toast.type === 'warning'" width="18" height="18" viewBox="0 0 18 18" fill="none">
            <path d="M8.134 2.5L1.5 14.5H16.5L9.866 2.5H8.134Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
            <path d="M9 7V10M9 12.5V12.51" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <svg v-else width="18" height="18" viewBox="0 0 18 18" fill="none">
            <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="1.5"/>
            <path d="M9 8V12.5M9 5.5V5.51" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </span>
        <span class="toast__message">{{ toast.message }}</span>
        <button class="toast__close" @click="removeToast(toast.id)" aria-label="Dismiss">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M10.5 3.5L3.5 10.5M3.5 3.5L10.5 10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </button>
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<script setup>
import { ref } from 'vue';

const toasts = ref([]);
let nextId = 0;

function addToast({ message, type = 'info', duration = 4000 }) {
  const id = ++nextId;
  toasts.value.push({ id, message, type });
  if (duration > 0) {
    setTimeout(() => removeToast(id), duration);
  }
}

function removeToast(id) {
  const idx = toasts.value.findIndex((t) => t.id === id);
  if (idx !== -1) toasts.value.splice(idx, 1);
}

defineExpose({ addToast, removeToast });
</script>

<style lang="scss" scoped>
.toast-container {
  position: fixed;
  top: $space-4;
  right: $space-4;
  z-index: $z-toast;
  display: flex;
  flex-direction: column;
  gap: $space-2;
  max-width: 420px;
  width: 100%;
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: $space-3;
  padding: $space-3 $space-4;
  border-radius: $border-radius-md;
  background: $color-neutral-0;
  border: 1px solid $border-color;
  box-shadow: $shadow-lg;
  pointer-events: all;
  font-size: $font-size-base;

  &--success {
    border-left: 3px solid $color-success-500;
    .toast__icon { color: $color-success-500; }
  }

  &--error {
    border-left: 3px solid $color-danger-500;
    .toast__icon { color: $color-danger-500; }
  }

  &--warning {
    border-left: 3px solid $color-warning-500;
    .toast__icon { color: $color-warning-500; }
  }

  &--info {
    border-left: 3px solid $color-primary-500;
    .toast__icon { color: $color-primary-500; }
  }

  &__icon {
    flex-shrink: 0;
    display: flex;
    margin-top: 1px;
  }

  &__message {
    flex: 1;
    color: $color-neutral-700;
    line-height: $line-height-base;
  }

  &__close {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: $border-radius-sm;
    color: $color-neutral-400;
    transition: all $transition-fast;

    &:hover {
      background: $color-neutral-50;
      color: $color-neutral-600;
    }
  }
}

.toast-enter-active {
  transition: all $transition-base;
}

.toast-leave-active {
  transition: all $transition-fast;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(24px);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(24px);
}

.toast-move {
  transition: transform $transition-base;
}
</style>
