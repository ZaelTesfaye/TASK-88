<template>
  <div v-if="!authStore.isAuthenticated" class="auth-layout">
    <router-view />
  </div>
  <div v-else class="app-layout">
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed, 'mobile-open': mobileMenuOpen }">
      <div class="sidebar-header">
        <div class="logo">
          <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
            <rect width="32" height="32" rx="8" fill="#1a73e8" />
            <path d="M8 12L16 8L24 12V20L16 24L8 20V12Z" stroke="#fff" stroke-width="1.5" stroke-linejoin="round" />
            <path d="M16 16V24" stroke="#fff" stroke-width="1.5" />
            <path d="M8 12L16 16L24 12" stroke="#fff" stroke-width="1.5" />
          </svg>
          <span v-if="!sidebarCollapsed" class="logo-text">Multi-Org Hub</span>
        </div>
      </div>
      <nav class="sidebar-nav">
        <router-link
          v-for="item in visibleNavItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          active-class="active"
          @click="mobileMenuOpen = false"
        >
          <span class="nav-icon" v-html="item.icon"></span>
          <span v-if="!sidebarCollapsed" class="nav-label">{{ item.label }}</span>
        </router-link>
      </nav>
      <button class="collapse-toggle" @click="sidebarCollapsed = !sidebarCollapsed">
        <svg
          width="18" height="18" viewBox="0 0 18 18" fill="none"
          :class="{ rotated: sidebarCollapsed }"
        >
          <path d="M11 4L6 9L11 14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
    </aside>

    <div v-if="mobileMenuOpen" class="mobile-overlay" @click="mobileMenuOpen = false"></div>

    <div class="main-area">
      <header class="topbar">
        <button class="hamburger" @click="mobileMenuOpen = !mobileMenuOpen">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M3 5H17M3 10H17M3 15H17" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
          </svg>
        </button>
        <AppBreadcrumb :items="contextStore.contextBreadcrumb" @navigate="onBreadcrumbNav" />
        <div class="topbar-spacer"></div>
        <div class="user-menu" :class="{ open: userMenuOpen }">
          <button class="user-menu-trigger" @click="userMenuOpen = !userMenuOpen">
            <div class="user-avatar">
              {{ userInitials }}
            </div>
            <span class="username">{{ authStore.user?.username }}</span>
            <AppChip :status="roleChipStatus" :label="roleDisplayLabel" size="sm" />
            <svg class="user-menu-chevron" width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M4 5.5L7 8.5L10 5.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </button>
          <div v-if="userMenuOpen" class="user-dropdown">
            <div class="user-dropdown-header">
              <span class="user-dropdown-name">{{ authStore.user?.username }}</span>
              <span class="user-dropdown-role">{{ roleDisplayLabel }}</span>
            </div>
            <div class="user-dropdown-divider"></div>
            <button class="user-dropdown-item user-dropdown-item--danger" @click="handleLogout">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M6 14H3.333A1.333 1.333 0 012 12.667V3.333A1.333 1.333 0 013.333 2H6M10.667 11.333L14 8M14 8L10.667 4.667M14 8H6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              Logout
            </button>
          </div>
        </div>
      </header>
      <main class="content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth.js';
import { useContextStore } from '@/stores/context.js';
import AppBreadcrumb from '@/components/common/AppBreadcrumb.vue';
import AppChip from '@/components/common/AppChip.vue';

const router = useRouter();
const authStore = useAuthStore();
const contextStore = useContextStore();

const sidebarCollapsed = ref(false);
const mobileMenuOpen = ref(false);
const userMenuOpen = ref(false);

const ROLE_LABELS = {
  system_admin: 'System Admin',
  data_steward: 'Data Steward',
  operations_analyst: 'Ops Analyst',
  standard_user: 'Standard',
  viewer: 'Viewer',
};

const ROLE_CHIP_STATUS = {
  system_admin: 'active',
  data_steward: 'review',
  operations_analyst: 'running',
  standard_user: 'inactive',
  viewer: 'inactive',
};

const navItems = [
  {
    path: '/org',
    label: 'Org Tree',
    icon: '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M10 2V6M10 6H6M10 6H14M6 6V10M14 6V10M6 10H3V14H7V10H6ZM14 10H11V14H15V10H14Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    roles: ['system_admin'],
  },
  {
    path: '/master/sku',
    label: 'Master Data',
    icon: '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><rect x="3" y="3" width="14" height="14" rx="2" stroke="currentColor" stroke-width="1.5"/><path d="M3 8H17M8 8V17" stroke="currentColor" stroke-width="1.5"/></svg>',
    roles: ['system_admin', 'data_steward', 'operations_analyst', 'standard_user'],
  },
  {
    path: '/playback',
    label: 'Playback',
    icon: '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><polygon points="6,3 18,10 6,17" stroke="currentColor" stroke-width="1.5" fill="none" stroke-linejoin="round"/></svg>',
    roles: ['system_admin', 'data_steward', 'operations_analyst', 'standard_user', 'viewer'],
  },
  {
    path: '/analytics',
    label: 'Analytics',
    icon: '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M3 17V11M8 17V7M13 17V10M18 17V4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
    roles: ['system_admin', 'operations_analyst'],
  },
  {
    path: '/ingestion',
    label: 'Ingestion',
    icon: '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M10 2V12M6 8L10 12L14 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M3 14V16C3 17.1 3.9 18 5 18H15C16.1 18 17 17.1 17 16V14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
    roles: ['system_admin', 'operations_analyst'],
  },
  {
    path: '/reports',
    label: 'Reports',
    icon: '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M12 2H5C3.9 2 3 2.9 3 4V16C3 17.1 3.9 18 5 18H15C16.1 18 17 17.1 17 16V7L12 2Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/><path d="M12 2V7H17" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/><path d="M7 10H13M7 13H11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
    roles: ['system_admin', 'operations_analyst'],
  },
  {
    path: '/security',
    label: 'Security Admin',
    icon: '<svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M10 2L3 5V9.09C3 13.14 5.87 16.92 10 18C14.13 16.92 17 13.14 17 9.09V5L10 2Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/><path d="M7 10L9 12L13 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    roles: ['system_admin'],
  },
];

const visibleNavItems = computed(() => {
  const role = authStore.userRole;
  if (!role) return [];
  return navItems.filter((item) => item.roles.includes(role));
});

const userInitials = computed(() => {
  const name = authStore.user?.username || '';
  return name.slice(0, 2).toUpperCase();
});

const roleDisplayLabel = computed(() => {
  return ROLE_LABELS[authStore.userRole] || authStore.userRole || '';
});

const roleChipStatus = computed(() => {
  return ROLE_CHIP_STATUS[authStore.userRole] || 'inactive';
});

function onBreadcrumbNav(item) {
  contextStore.switchContext(item.id);
}

async function handleLogout() {
  userMenuOpen.value = false;
  await authStore.logout();
}

function onClickOutsideUserMenu(e) {
  const menu = document.querySelector('.user-menu');
  if (menu && !menu.contains(e.target)) {
    userMenuOpen.value = false;
  }
}

onMounted(() => {
  document.addEventListener('click', onClickOutsideUserMenu);
});

onUnmounted(() => {
  document.removeEventListener('click', onClickOutsideUserMenu);
});
</script>

<style lang="scss" scoped>
.auth-layout {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-layout {
  display: flex;
  min-height: 100vh;
}

// Sidebar
.sidebar {
  width: 240px;
  background: #1e293b;
  color: #fff;
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: $z-sidebar;
  transition: width $transition-slow;

  &.collapsed {
    width: 64px;

    .sidebar-nav .nav-item {
      justify-content: center;
      padding: $space-3;
    }
  }
}

.sidebar-header {
  height: $topbar-height;
  display: flex;
  align-items: center;
  padding: 0 $space-4;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
}

.logo {
  display: flex;
  align-items: center;
  gap: $space-3;
  overflow: hidden;

  svg {
    flex-shrink: 0;
  }
}

.logo-text {
  font-size: $font-size-md;
  font-weight: $font-weight-bold;
  white-space: nowrap;
  letter-spacing: -0.3px;
}

.sidebar-nav {
  flex: 1;
  padding: $space-3;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: $space-3;
  padding: $space-2 $space-3;
  border-radius: $border-radius-base;
  color: rgba(255, 255, 255, 0.6);
  font-size: $font-size-base;
  font-weight: $font-weight-medium;
  text-decoration: none;
  transition: all $transition-fast;
  white-space: nowrap;
  overflow: hidden;

  &:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.08);
    text-decoration: none;
  }

  &.active {
    color: #fff;
    background: rgba($color-primary-500, 0.3);

    .nav-icon {
      color: $color-primary-300;
    }
  }
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
}

.nav-label {
  overflow: hidden;
  text-overflow: ellipsis;
}

.collapse-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  margin: $space-3;
  padding: $space-2;
  border-radius: $border-radius-base;
  color: rgba(255, 255, 255, 0.4);
  transition: all $transition-fast;
  flex-shrink: 0;

  &:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.08);
  }

  svg {
    transition: transform $transition-base;
  }

  .rotated {
    transform: rotate(180deg);
  }
}

// Main Area
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  margin-left: 240px;
  transition: margin-left $transition-slow;

  .sidebar.collapsed ~ & {
    margin-left: 64px;
  }
}

.collapsed ~ .main-area {
  margin-left: 64px;
}

// Top Bar
.topbar {
  height: $topbar-height;
  display: flex;
  align-items: center;
  gap: $space-4;
  padding: 0 $space-6;
  background: $color-neutral-0;
  border-bottom: 1px solid $border-color;
  position: sticky;
  top: 0;
  z-index: $z-topbar;
  flex-shrink: 0;
}

.hamburger {
  display: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: $border-radius-base;
  color: $color-neutral-600;
  transition: all $transition-fast;

  &:hover {
    background: $color-neutral-50;
  }
}

.topbar-spacer {
  flex: 1;
}

// User Menu
.user-menu {
  position: relative;
}

.user-menu-trigger {
  display: flex;
  align-items: center;
  gap: $space-2;
  padding: $space-1 $space-2;
  border-radius: $border-radius-base;
  transition: background $transition-fast;

  &:hover {
    background: $color-neutral-50;
  }
}

.user-avatar {
  width: 30px;
  height: 30px;
  border-radius: $border-radius-full;
  background: $color-primary-500;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: $font-size-xs;
  font-weight: $font-weight-bold;
  flex-shrink: 0;
}

.username {
  font-size: $font-size-base;
  font-weight: $font-weight-medium;
  color: $color-neutral-700;
}

.user-menu-chevron {
  color: $color-neutral-400;
  transition: transform $transition-fast;

  .open & {
    transform: rotate(180deg);
  }
}

.user-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  min-width: 200px;
  background: $color-neutral-0;
  border: 1px solid $border-color;
  border-radius: $border-radius-md;
  box-shadow: $shadow-lg;
  z-index: $z-dropdown;
  animation: dropdown-in $transition-fast forwards;
}

.user-dropdown-header {
  padding: $space-3 $space-4;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-dropdown-name {
  font-size: $font-size-base;
  font-weight: $font-weight-semibold;
  color: $color-neutral-800;
}

.user-dropdown-role {
  font-size: $font-size-sm;
  color: $color-neutral-500;
}

.user-dropdown-divider {
  border-top: 1px solid $border-color;
}

.user-dropdown-item {
  display: flex;
  align-items: center;
  gap: $space-2;
  width: 100%;
  padding: $space-2 $space-4;
  font-size: $font-size-base;
  color: $color-neutral-700;
  transition: background $transition-fast;

  &:hover {
    background: $color-neutral-50;
  }

  &--danger {
    color: $color-danger-500;

    &:hover {
      background: $color-danger-50;
    }
  }
}

@keyframes dropdown-in {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

// Content
.content {
  flex: 1;
  padding: $space-6;
  background: #f8fafc;
  min-height: calc(100vh - #{$topbar-height});
}

// Mobile overlay
.mobile-overlay {
  display: none;
}

// Responsive
@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);

    &.mobile-open {
      transform: translateX(0);
    }
  }

  .main-area {
    margin-left: 0 !important;
  }

  .hamburger {
    display: flex;
  }

  .mobile-overlay {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    z-index: $z-sidebar - 1;
  }

  .collapse-toggle {
    display: none;
  }

  .username {
    display: none;
  }

  .content {
    padding: $space-4;
  }
}
</style>
