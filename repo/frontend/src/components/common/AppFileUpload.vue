<template>
  <div
    class="app-file-upload"
    :class="{
      'app-file-upload--dragging': isDragging,
      'app-file-upload--error': !!error,
      'app-file-upload--disabled': disabled,
    }"
    @dragover.prevent="onDragOver"
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
    @click="openFilePicker"
  >
    <input
      ref="fileInput"
      type="file"
      class="app-file-upload__input"
      :accept="accept"
      :multiple="multiple"
      :disabled="disabled"
      @change="onFileSelect"
    />

    <div v-if="!selectedFile" class="app-file-upload__placeholder">
      <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
        <path d="M16 6V22M10 12L16 6L22 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M4 22V24C4 25.1046 4.89543 26 6 26H26C27.1046 26 28 25.1046 28 24V22" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      <p class="app-file-upload__text">
        <span class="app-file-upload__link">Click to upload</span> or drag and drop
      </p>
      <p v-if="hint" class="app-file-upload__hint">{{ hint }}</p>
    </div>

    <div v-else class="app-file-upload__selected">
      <div class="app-file-upload__file-info">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path d="M11 2H5C3.89543 2 3 2.89543 3 4V16C3 17.1046 3.89543 18 5 18H15C16.1046 18 17 17.1046 17 16V8L11 2Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M11 2V8H17" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <div>
          <p class="app-file-upload__filename">{{ selectedFile.name }}</p>
          <p class="app-file-upload__filesize">{{ formatSize(selectedFile.size) }}</p>
        </div>
      </div>
      <button
        class="app-file-upload__remove"
        @click.stop="clearFile"
        aria-label="Remove file"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <path d="M12 4L4 12M4 4L12 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
    </div>

    <div v-if="progress !== null && progress >= 0" class="app-file-upload__progress">
      <div class="app-file-upload__progress-bar" :style="{ width: `${progress}%` }" />
    </div>
  </div>
  <p v-if="error" class="app-file-upload__error">{{ error }}</p>
</template>

<script setup>
import { ref } from 'vue';

const props = defineProps({
  accept: {
    type: String,
    default: '',
  },
  maxSize: {
    type: Number,
    default: 10 * 1024 * 1024, // 10 MB
  },
  multiple: Boolean,
  disabled: Boolean,
  hint: String,
  progress: {
    type: Number,
    default: null,
  },
});

const emit = defineEmits(['file-selected', 'file-removed', 'error']);

const fileInput = ref(null);
const selectedFile = ref(null);
const isDragging = ref(false);
const error = ref('');

function openFilePicker() {
  if (!props.disabled) {
    fileInput.value?.click();
  }
}

function validateFile(file) {
  if (props.accept) {
    const allowed = props.accept.split(',').map((s) => s.trim().toLowerCase());
    const ext = '.' + file.name.split('.').pop().toLowerCase();
    const matchesType = allowed.some(
      (a) => a === ext || a === file.type || (a.endsWith('/*') && file.type.startsWith(a.replace('/*', '/')))
    );
    if (!matchesType) {
      return `File type not allowed. Accepted: ${props.accept}`;
    }
  }
  if (file.size > props.maxSize) {
    return `File too large. Maximum: ${formatSize(props.maxSize)}`;
  }
  return null;
}

function handleFile(file) {
  const err = validateFile(file);
  if (err) {
    error.value = err;
    emit('error', err);
    return;
  }
  error.value = '';
  selectedFile.value = file;
  emit('file-selected', file);
}

function onFileSelect(event) {
  const file = event.target.files?.[0];
  if (file) handleFile(file);
}

function onDragOver() {
  if (!props.disabled) isDragging.value = true;
}

function onDragLeave() {
  isDragging.value = false;
}

function onDrop(event) {
  isDragging.value = false;
  if (props.disabled) return;
  const file = event.dataTransfer.files?.[0];
  if (file) handleFile(file);
}

function clearFile() {
  selectedFile.value = null;
  error.value = '';
  if (fileInput.value) fileInput.value.value = '';
  emit('file-removed');
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}
</script>

<style lang="scss" scoped>
.app-file-upload {
  border: 2px dashed $border-color;
  border-radius: $border-radius-md;
  padding: $space-6;
  text-align: center;
  cursor: pointer;
  transition: all $transition-fast;
  position: relative;
  background: $color-neutral-0;

  &:hover:not(.app-file-upload--disabled) {
    border-color: $color-primary-300;
    background: $color-primary-50;
  }

  &--dragging {
    border-color: $color-primary-500;
    background: $color-primary-50;
  }

  &--error {
    border-color: $color-danger-500;
  }

  &--disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &__input {
    position: absolute;
    inset: 0;
    opacity: 0;
    cursor: pointer;
    width: 100%;
    height: 100%;

    &:disabled {
      cursor: not-allowed;
    }
  }

  &__placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: $space-2;
    color: $color-neutral-400;
    pointer-events: none;
  }

  &__text {
    font-size: $font-size-base;
    color: $color-neutral-600;
    margin: 0;
  }

  &__link {
    color: $color-primary-500;
    font-weight: $font-weight-medium;
  }

  &__hint {
    font-size: $font-size-xs;
    color: $color-neutral-400;
    margin: 0;
  }

  &__selected {
    display: flex;
    align-items: center;
    justify-content: space-between;
    pointer-events: none;
  }

  &__file-info {
    display: flex;
    align-items: center;
    gap: $space-3;
    color: $color-neutral-600;
    text-align: left;
  }

  &__filename {
    font-size: $font-size-base;
    font-weight: $font-weight-medium;
    color: $color-neutral-800;
    margin: 0;
  }

  &__filesize {
    font-size: $font-size-xs;
    color: $color-neutral-400;
    margin: 0;
  }

  &__remove {
    pointer-events: all;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: $border-radius-base;
    color: $color-neutral-400;
    transition: all $transition-fast;

    &:hover {
      background: $color-danger-50;
      color: $color-danger-500;
    }
  }

  &__progress {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: $color-neutral-100;
    border-radius: 0 0 $border-radius-md $border-radius-md;
    overflow: hidden;
  }

  &__progress-bar {
    height: 100%;
    background: $color-primary-500;
    border-radius: 0 0 $border-radius-md $border-radius-md;
    transition: width $transition-base;
  }

  &__error {
    font-size: $font-size-xs;
    color: $color-danger-500;
    margin: $space-1 0 0;
  }
}
</style>
