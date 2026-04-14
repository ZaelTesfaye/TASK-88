<template>
  <div class="app-select" :class="{ 'app-select--error': !!error, 'app-select--disabled': disabled }">
    <label v-if="label" class="app-select__label">
      {{ label }}
      <span v-if="required" class="app-select__required">*</span>
    </label>
    <div class="app-select__wrapper">
      <select
        class="app-select__field"
        :value="modelValue"
        :disabled="disabled"
        @change="$emit('update:modelValue', $event.target.value)"
      >
        <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
        <option
          v-for="opt in normalizedOptions"
          :key="opt.value"
          :value="opt.value"
          :disabled="opt.disabled"
        >
          {{ opt.label }}
        </option>
      </select>
      <span class="app-select__chevron">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <path d="M4 6L8 10L12 6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </span>
    </div>
    <p v-if="error" class="app-select__error">{{ error }}</p>
    <p v-else-if="hint" class="app-select__hint">{{ hint }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  options: {
    type: Array,
    default: () => [],
  },
  label: String,
  placeholder: String,
  error: String,
  hint: String,
  required: Boolean,
  disabled: Boolean,
});

defineEmits(['update:modelValue']);

const normalizedOptions = computed(() =>
  props.options.map((opt) =>
    typeof opt === 'string' || typeof opt === 'number'
      ? { value: opt, label: String(opt) }
      : opt
  )
);
</script>

<style lang="scss" scoped>
.app-select {
  display: flex;
  flex-direction: column;
  gap: $space-1;

  &__label {
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    color: $color-neutral-700;
    user-select: none;
  }

  &__required {
    color: $color-danger-500;
    margin-left: 2px;
  }

  &__wrapper {
    position: relative;
  }

  &__field {
    width: 100%;
    height: 36px;
    padding: 0 36px 0 12px;
    border: 1px solid $border-color;
    border-radius: $border-radius-base;
    font-size: $font-size-base;
    color: $color-neutral-800;
    background: $color-neutral-0;
    outline: none;
    appearance: none;
    cursor: pointer;
    transition: border-color $transition-fast, box-shadow $transition-fast;

    &:hover:not(:disabled):not(:focus) {
      border-color: $border-color-hover;
    }

    &:focus {
      border-color: $color-primary-500;
      box-shadow: 0 0 0 3px rgba($color-primary-500, 0.12);
    }

    &:disabled {
      background: $color-neutral-50;
      color: $color-neutral-400;
      cursor: not-allowed;
    }
  }

  &__chevron {
    position: absolute;
    right: 10px;
    top: 50%;
    transform: translateY(-50%);
    color: $color-neutral-400;
    pointer-events: none;
    display: flex;
  }

  &--error {
    .app-select__field {
      border-color: $color-danger-500;

      &:focus {
        box-shadow: 0 0 0 3px rgba($color-danger-500, 0.12);
      }
    }
  }

  &__error {
    font-size: $font-size-xs;
    color: $color-danger-500;
    margin: 0;
  }

  &__hint {
    font-size: $font-size-xs;
    color: $color-neutral-400;
    margin: 0;
  }
}
</style>
