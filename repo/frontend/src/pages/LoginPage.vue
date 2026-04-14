<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <svg class="login-logo" width="48" height="48" viewBox="0 0 48 48" fill="none">
          <rect width="48" height="48" rx="12" fill="#1a73e8" />
          <path d="M12 18L24 12L36 18V30L24 36L12 30V18Z" stroke="#fff" stroke-width="2" stroke-linejoin="round" />
          <path d="M24 24V36" stroke="#fff" stroke-width="2" />
          <path d="M12 18L24 24L36 18" stroke="#fff" stroke-width="2" />
        </svg>
        <h1 class="login-title">Multi-Org Hub</h1>
        <p class="login-subtitle">Sign in to your account</p>
      </div>

      <div v-if="errorMessage" class="login-alert" :class="`login-alert--${errorType}`">
        <svg v-if="errorType === 'locked'" width="18" height="18" viewBox="0 0 18 18" fill="none">
          <rect x="3" y="8" width="12" height="8" rx="2" stroke="currentColor" stroke-width="1.5" />
          <path d="M6 8V5a3 3 0 016 0v3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        </svg>
        <svg v-else width="18" height="18" viewBox="0 0 18 18" fill="none">
          <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="1.5" />
          <path d="M9 5.5V9.5M9 12V12.01" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        </svg>
        <span>{{ errorMessage }}</span>
      </div>

      <form class="login-form" @submit.prevent="handleLogin" novalidate>
        <div class="form-field">
          <label class="form-field__label" for="login-username">Username</label>
          <div class="form-field__wrapper" :class="{ 'form-field__wrapper--error': fieldErrors.username }">
            <svg class="form-field__icon" width="18" height="18" viewBox="0 0 18 18" fill="none">
              <circle cx="9" cy="5.5" r="3.5" stroke="currentColor" stroke-width="1.5" />
              <path d="M2 16C2 13.24 4.24 11 7 11H11C13.76 11 16 13.24 16 16" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
            <input
              id="login-username"
              v-model.trim="username"
              type="text"
              class="form-field__input"
              placeholder="Enter your username"
              autocomplete="username"
              :disabled="loading"
              @input="clearFieldError('username')"
            />
          </div>
          <span v-if="fieldErrors.username" class="form-field__error">{{ fieldErrors.username }}</span>
        </div>

        <div class="form-field">
          <label class="form-field__label" for="login-password">Password</label>
          <div class="form-field__wrapper" :class="{ 'form-field__wrapper--error': fieldErrors.password }">
            <svg class="form-field__icon" width="18" height="18" viewBox="0 0 18 18" fill="none">
              <rect x="3" y="8" width="12" height="8" rx="2" stroke="currentColor" stroke-width="1.5" />
              <path d="M6 8V5a3 3 0 016 0v3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
            <input
              id="login-password"
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              class="form-field__input"
              placeholder="Enter your password"
              autocomplete="current-password"
              :disabled="loading"
              @input="clearFieldError('password')"
              @keydown.enter="handleLogin"
            />
            <button
              type="button"
              class="form-field__toggle"
              tabindex="-1"
              @click="showPassword = !showPassword"
              :aria-label="showPassword ? 'Hide password' : 'Show password'"
            >
              <svg v-if="!showPassword" width="18" height="18" viewBox="0 0 18 18" fill="none">
                <path d="M1.5 9S4.5 3 9 3s7.5 6 7.5 6-3 6-7.5 6S1.5 9 1.5 9z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                <circle cx="9" cy="9" r="2.5" stroke="currentColor" stroke-width="1.5" />
              </svg>
              <svg v-else width="18" height="18" viewBox="0 0 18 18" fill="none">
                <path d="M2.5 2.5L15.5 15.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                <path d="M7.58 7.58a2.5 2.5 0 003.34 3.34" stroke="currentColor" stroke-width="1.5" />
                <path d="M4.18 4.64C2.76 5.89 1.5 9 1.5 9s3 6 7.5 6c1.4 0 2.68-.5 3.78-1.25" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                <path d="M14.77 12.03C15.86 10.82 16.5 9 16.5 9s-3-6-7.5-6c-.47 0-.93.05-1.37.14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
              </svg>
            </button>
          </div>
          <span v-if="fieldErrors.password" class="form-field__error">{{ fieldErrors.password }}</span>
          <p class="form-field__hint">Min 12 characters with uppercase, lowercase, digit, and symbol</p>
        </div>

        <button type="submit" class="login-btn" :disabled="loading">
          <svg v-if="loading" class="login-btn__spinner" width="18" height="18" viewBox="0 0 18 18" fill="none">
            <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="2" opacity="0.3" />
            <path d="M9 2a7 7 0 017 7" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          </svg>
          <span :class="{ invisible: loading }">Sign In</span>
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/auth.js';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const username = ref('');
const password = ref('');
const showPassword = ref(false);
const loading = ref(false);
const errorMessage = ref('');
const errorType = ref('error');
const fieldErrors = reactive({ username: '', password: '' });

function clearFieldError(field) {
  fieldErrors[field] = '';
  errorMessage.value = '';
}

function validate() {
  let valid = true;
  fieldErrors.username = '';
  fieldErrors.password = '';

  if (!username.value) {
    fieldErrors.username = 'Username is required';
    valid = false;
  }

  if (!password.value) {
    fieldErrors.password = 'Password is required';
    valid = false;
  } else if (password.value.length < 12) {
    fieldErrors.password = 'Password must be at least 12 characters';
    valid = false;
  } else {
    const hasUpper = /[A-Z]/.test(password.value);
    const hasLower = /[a-z]/.test(password.value);
    const hasDigit = /\d/.test(password.value);
    const hasSymbol = /[^A-Za-z0-9]/.test(password.value);
    if (!hasUpper || !hasLower || !hasDigit || !hasSymbol) {
      fieldErrors.password = 'Must include uppercase, lowercase, digit, and symbol';
      valid = false;
    }
  }

  return valid;
}

async function handleLogin() {
  errorMessage.value = '';
  if (!validate()) return;

  loading.value = true;
  try {
    await authStore.login(username.value, password.value);
    const redirect = route.query.redirect || '/master/sku';
    router.push(redirect);
  } catch (err) {
    const status = err?.response?.status;
    const data = err?.response?.data;

    if (status === 401) {
      errorType.value = 'error';
      errorMessage.value = 'Invalid username or password. Please try again.';
    } else if (status === 423) {
      errorType.value = 'locked';
      const until = data?.lockedUntil;
      if (until) {
        const time = new Date(until).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        errorMessage.value = `Account locked until ${time}. Too many failed attempts.`;
      } else {
        errorMessage.value = 'Account is temporarily locked. Please try again later.';
      }
    } else {
      errorType.value = 'error';
      errorMessage.value = data?.message || 'Unable to sign in. Please check your connection and try again.';
    }
  } finally {
    loading.value = false;
  }
}
</script>

<style lang="scss" scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $space-4;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);

  &::before {
    content: '';
    position: fixed;
    inset: 0;
    background:
      radial-gradient(ellipse at 20% 50%, rgba($color-primary-500, 0.12) 0%, transparent 50%),
      radial-gradient(ellipse at 80% 20%, rgba($color-primary-500, 0.08) 0%, transparent 50%);
    pointer-events: none;
  }
}

.login-card {
  position: relative;
  width: 100%;
  max-width: 420px;
  background: $color-neutral-0;
  border-radius: $border-radius-lg;
  box-shadow: $shadow-xl, 0 0 0 1px rgba(255, 255, 255, 0.05);
  padding: $space-8 $space-7;
  animation: card-in 400ms ease forwards;
}

@keyframes card-in {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.login-header {
  text-align: center;
  margin-bottom: $space-7;
}

.login-logo {
  margin: 0 auto $space-4;
}

.login-title {
  font-size: $font-size-2xl;
  font-weight: $font-weight-bold;
  color: $color-neutral-900;
  margin-bottom: $space-1;
}

.login-subtitle {
  font-size: $font-size-base;
  color: $color-neutral-500;
}

.login-alert {
  display: flex;
  align-items: flex-start;
  gap: $space-3;
  padding: $space-3 $space-4;
  border-radius: $border-radius-base;
  font-size: $font-size-base;
  margin-bottom: $space-5;
  animation: alert-in 200ms ease forwards;

  svg {
    flex-shrink: 0;
    margin-top: 1px;
  }

  &--error {
    background: $color-danger-50;
    color: $color-danger-600;
    border: 1px solid rgba($color-danger-500, 0.2);
  }

  &--locked {
    background: $color-warning-50;
    color: $color-warning-700;
    border: 1px solid rgba($color-warning-500, 0.2);
  }
}

@keyframes alert-in {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: $space-5;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: $space-1;

  &__label {
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    color: $color-neutral-700;
  }

  &__wrapper {
    position: relative;
    display: flex;
    align-items: center;
    border: 1px solid $border-color;
    border-radius: $border-radius-base;
    background: $color-neutral-0;
    transition: border-color $transition-fast, box-shadow $transition-fast;

    &:focus-within {
      border-color: $color-primary-500;
      box-shadow: 0 0 0 3px rgba($color-primary-500, 0.12);
    }

    &--error {
      border-color: $color-danger-500;

      &:focus-within {
        box-shadow: 0 0 0 3px rgba($color-danger-500, 0.12);
      }
    }
  }

  &__icon {
    position: absolute;
    left: 12px;
    color: $color-neutral-400;
    pointer-events: none;
    flex-shrink: 0;
  }

  &__input {
    width: 100%;
    height: 42px;
    padding: 0 40px 0 38px;
    border: none;
    border-radius: $border-radius-base;
    font-size: $font-size-base;
    color: $color-neutral-800;
    background: transparent;
    outline: none;

    &::placeholder {
      color: $color-neutral-300;
    }

    &:disabled {
      color: $color-neutral-400;
      cursor: not-allowed;
    }
  }

  &__toggle {
    position: absolute;
    right: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border-radius: $border-radius-sm;
    color: $color-neutral-400;
    transition: color $transition-fast, background $transition-fast;

    &:hover {
      color: $color-neutral-600;
      background: $color-neutral-50;
    }
  }

  &__error {
    font-size: $font-size-xs;
    color: $color-danger-500;
  }

  &__hint {
    font-size: $font-size-xs;
    color: $color-neutral-400;
    margin: 2px 0 0;
  }
}

.login-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 44px;
  border: none;
  border-radius: $border-radius-base;
  background: $color-primary-500;
  color: #fff;
  font-size: $font-size-md;
  font-weight: $font-weight-semibold;
  cursor: pointer;
  transition: background $transition-fast, transform $transition-fast;
  margin-top: $space-2;

  &:hover:not(:disabled) {
    background: $color-primary-600;
  }

  &:active:not(:disabled) {
    background: $color-primary-700;
    transform: scale(0.99);
  }

  &:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  &__spinner {
    position: absolute;
    animation: spin 0.8s linear infinite;
  }

  .invisible {
    visibility: hidden;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
