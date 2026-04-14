<template>
  <div class="app-input" :class="{ 'app-input--error': !!error, 'app-input--disabled': disabled }">
    <label v-if="label" class="app-input__label" :for="inputId">
      {{ label }}
      <span v-if="required" class="app-input__required">*</span>
    </label>
    <div class="app-input__wrapper">
      <span v-if="$slots.prefix" class="app-input__prefix">
        <slot name="prefix" />
      </span>
      <input
        v-if="type !== 'textarea'"
        :id="inputId"
        class="app-input__field"
        :class="{ 'app-input__field--has-prefix': $slots.prefix, 'app-input__field--has-suffix': $slots.suffix }"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        :min="min"
        :max="max"
        :step="step"
        :autocomplete="autocomplete"
        @input="$emit('update:modelValue', $event.target.value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
      />
      <textarea
        v-else
        :id="inputId"
        class="app-input__field app-input__field--textarea"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        :rows="rows"
        @input="$emit('update:modelValue', $event.target.value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
      />
      <span v-if="$slots.suffix" class="app-input__suffix">
        <slot name="suffix" />
      </span>
    </div>
    <p v-if="error" class="app-input__error">{{ error }}</p>
    <p v-else-if="hint" class="app-input__hint">{{ hint }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  label: String,
  placeholder: String,
  type: {
    type: String,
    default: 'text',
  },
  error: String,
  hint: String,
  required: Boolean,
  disabled: Boolean,
  readonly: Boolean,
  rows: {
    type: Number,
    default: 3,
  },
  min: [String, Number],
  max: [String, Number],
  step: [String, Number],
  autocomplete: String,
});

defineEmits(['update:modelValue', 'blur', 'focus']);

let idCounter = 0;
const inputId = computed(() => `app-input-${++idCounter}`);
</script>

<style lang="scss" scoped>
.app-input {
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
    display: flex;
    align-items: center;
  }

  &__prefix,
  &__suffix {
    position: absolute;
    display: flex;
    align-items: center;
    color: $color-neutral-400;
    pointer-events: none;
  }

  &__prefix {
    left: 12px;
  }

  &__suffix {
    right: 12px;
  }

  &__field {
    width: 100%;
    height: 36px;
    padding: 0 12px;
    border: 1px solid $border-color;
    border-radius: $border-radius-base;
    font-size: $font-size-base;
    color: $color-neutral-800;
    background: $color-neutral-0;
    outline: none;
    transition: border-color $transition-fast, box-shadow $transition-fast;

    &::placeholder {
      color: $color-neutral-300;
    }

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

    &--has-prefix {
      padding-left: 36px;
    }

    &--has-suffix {
      padding-right: 36px;
    }

    &--textarea {
      height: auto;
      padding: 8px 12px;
      resize: vertical;
      font-family: inherit;
    }
  }

  &--error {
    .app-input__field {
      border-color: $color-danger-500;

      &:focus {
        box-shadow: 0 0 0 3px rgba($color-danger-500, 0.12);
      }
    }
  }

  &--disabled {
    opacity: 0.6;
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
