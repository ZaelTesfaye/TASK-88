import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import * as authApi from '@/api/auth.js';
import { setLogoutCallback } from '@/api/client.js';
import router from '@/router/index.js';

const IDLE_TIMEOUT_MS = 30 * 60 * 1000; // 30 minutes

export const useAuthStore = defineStore('auth', () => {
  // ---- State ----
  const user = ref(null);
  const token = ref(null);
  const refreshToken = ref(null);
  const currentContext = ref(null);
  let idleTimer = null;
  let lastActivity = Date.now();

  // ---- Getters ----
  const isAuthenticated = computed(() => !!token.value && !!user.value);

  const userRole = computed(() => user.value?.role || null);

  const permissions = computed(() => user.value?.permissions || []);

  const currentScope = computed(() => currentContext.value?.id || null);

  // ---- Helpers ----
  function persistAuth() {
    if (token.value) {
      localStorage.setItem('auth_token', token.value);
    }
    if (refreshToken.value) {
      localStorage.setItem('auth_refresh_token', refreshToken.value);
    }
    if (user.value) {
      localStorage.setItem('auth_user', JSON.stringify(user.value));
    }
  }

  function clearAuth() {
    user.value = null;
    token.value = null;
    refreshToken.value = null;
    currentContext.value = null;
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_refresh_token');
    localStorage.removeItem('auth_user');
    stopIdleTracking();
  }

  function resetIdleTimer() {
    lastActivity = Date.now();
  }

  function startIdleTracking() {
    stopIdleTracking();
    const events = ['mousedown', 'keydown', 'scroll', 'touchstart'];
    events.forEach((evt) => window.addEventListener(evt, resetIdleTimer, { passive: true }));
    idleTimer = setInterval(() => {
      if (Date.now() - lastActivity >= IDLE_TIMEOUT_MS) {
        logout();
      }
    }, 60000); // check every minute
  }

  function stopIdleTracking() {
    if (idleTimer) {
      clearInterval(idleTimer);
      idleTimer = null;
    }
    const events = ['mousedown', 'keydown', 'scroll', 'touchstart'];
    events.forEach((evt) => window.removeEventListener(evt, resetIdleTimer));
  }

  // ---- Actions ----
  async function login(username, password) {
    const { data } = await authApi.login(username, password);
    token.value = data.token;
    refreshToken.value = data.refreshToken;
    user.value = data.user;
    currentContext.value = data.defaultContext || null;
    persistAuth();
    startIdleTracking();
  }

  async function logout() {
    try {
      if (token.value) {
        await authApi.logout();
      }
    } catch {
      // ignore errors on logout
    } finally {
      clearAuth();
      router.push('/login');
    }
  }

  async function refreshSession() {
    if (!refreshToken.value) {
      clearAuth();
      return;
    }
    try {
      const { data } = await authApi.refresh(refreshToken.value);
      token.value = data.token;
      refreshToken.value = data.refreshToken;
      persistAuth();
    } catch {
      clearAuth();
      router.push('/login');
    }
  }

  function switchContext(orgNode) {
    currentContext.value = orgNode;
    persistAuth();
  }

  function restoreSession() {
    const savedToken = localStorage.getItem('auth_token');
    const savedRefresh = localStorage.getItem('auth_refresh_token');
    const savedUser = localStorage.getItem('auth_user');

    if (savedToken && savedUser) {
      token.value = savedToken;
      refreshToken.value = savedRefresh;
      try {
        user.value = JSON.parse(savedUser);
      } catch {
        clearAuth();
        return;
      }
      startIdleTracking();
    }
  }

  function hasRole(role) {
    return userRole.value === role;
  }

  function hasAnyRole(roles) {
    return roles.includes(userRole.value);
  }

  // Wire up auto-logout on 401
  setLogoutCallback(() => {
    clearAuth();
    router.push('/login');
  });

  // Restore on creation
  restoreSession();

  return {
    user,
    token,
    refreshToken,
    currentContext,
    isAuthenticated,
    userRole,
    permissions,
    currentScope,
    login,
    logout,
    refreshSession,
    switchContext,
    restoreSession,
    hasRole,
    hasAnyRole,
  };
});
