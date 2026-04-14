<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="modelValue" class="dialog-overlay" @click.self="onOverlayClick">
        <div
          class="dialog"
          :class="[`dialog--${size}`, { 'dialog--danger': danger }]"
          role="dialog"
          :aria-label="title"
          aria-modal="true"
        >
          <div class="dialog__header">
            <h3 class="dialog__title">{{ title }}</h3>
            <button
              class="dialog__close"
              aria-label="Close dialog"
              @click="close"
            >
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path d="M15 5L5 15M5 5l10 10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
              </svg>
            </button>
          </div>

          <div class="dialog__body">
            <slot />
          </div>

          <div v-if="$slots.footer" class="dialog__footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
const props = defineProps({
  modelValue: Boolean,
  title: {
    type: String,
    default: '',
  },
  size: {
    type: String,
    default: 'md',
    validator: (v) => ['sm', 'md', 'lg', 'xl'].includes(v),
  },
  persistent: Boolean,
  danger: Boolean,
});

const emit = defineEmits(['update:modelValue', 'close']);

function close() {
  emit('update:modelValue', false);
  emit('close');
}

function onOverlayClick() {
  if (!props.persistent) {
    close();
  }
}
</script>

<style lang="scss" scoped>
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: $z-dialog-overlay;
  padding: $space-4;
}

.dialog {
  background: $color-neutral-0;
  border-radius: $border-radius-lg;
  box-shadow: $shadow-dialog;
  max-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: dialog-in 200ms ease forwards;

  &--sm { width: 400px; }
  &--md { width: 540px; }
  &--lg { width: 720px; }
  &--xl { width: 960px; }

  &--danger {
    .dialog__title {
      color: $color-danger-600;
    }
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: $space-5 $space-6;
    border-bottom: 1px solid $border-color;
    flex-shrink: 0;
  }

  &__title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    color: $color-neutral-900;
    margin: 0;
  }

  &__close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: $border-radius-base;
    color: $color-neutral-400;
    transition: all $transition-fast;

    &:hover {
      background: $color-neutral-50;
      color: $color-neutral-600;
    }
  }

  &__body {
    padding: $space-6;
    overflow-y: auto;
    flex: 1;
  }

  &__footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: $space-3;
    padding: $space-4 $space-6;
    border-top: 1px solid $border-color;
    flex-shrink: 0;
  }
}

@keyframes dialog-in {
  from {
    opacity: 0;
    transform: scale(0.95) translateY(8px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.dialog-enter-active {
  transition: opacity 200ms ease;
}

.dialog-leave-active {
  transition: opacity 150ms ease;
}

.dialog-enter-from,
.dialog-leave-to {
  opacity: 0;
}
</style>
