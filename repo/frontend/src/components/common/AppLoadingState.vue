<template>
  <div class="app-loading" :class="{ 'app-loading--overlay': overlay }">
    <div v-if="variant === 'spinner'" class="app-loading__spinner">
      <svg :width="size" :height="size" viewBox="0 0 40 40" fill="none">
        <circle cx="20" cy="20" r="16" stroke="currentColor" stroke-width="3" opacity="0.2" />
        <path
          d="M20 4a16 16 0 0 1 16 16"
          stroke="currentColor"
          stroke-width="3"
          stroke-linecap="round"
        />
      </svg>
      <p v-if="message" class="app-loading__message">{{ message }}</p>
    </div>
    <div v-else class="app-loading__skeleton">
      <div
        v-for="n in lines"
        :key="n"
        class="skeleton"
        :style="{
          width: n === lines ? '60%' : '100%',
          height: '14px',
          marginBottom: '10px',
        }"
      />
    </div>
  </div>
</template>

<script setup>
defineProps({
  variant: {
    type: String,
    default: 'spinner',
    validator: (v) => ['spinner', 'skeleton'].includes(v),
  },
  size: {
    type: Number,
    default: 40,
  },
  message: String,
  overlay: Boolean,
  lines: {
    type: Number,
    default: 4,
  },
});
</script>

<style lang="scss" scoped>
.app-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $space-8;

  &--overlay {
    position: absolute;
    inset: 0;
    background: rgba(255, 255, 255, 0.8);
    z-index: 10;
    border-radius: inherit;
  }

  &__spinner {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: $space-3;
    color: $color-primary-500;

    svg {
      animation: spin 1s linear infinite;
    }
  }

  &__message {
    font-size: $font-size-sm;
    color: $color-neutral-500;
    margin: 0;
  }

  &__skeleton {
    width: 100%;
    max-width: 500px;
  }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
