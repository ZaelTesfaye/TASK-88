import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '@/stores/auth.js';

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/LoginPage.vue'),
    meta: { public: true },
  },
  {
    path: '/org',
    name: 'OrgTree',
    component: () => import('@/pages/OrgTreePage.vue'),
    meta: { roles: ['system_admin'] },
  },
  {
    path: '/master/:entity',
    name: 'MasterData',
    component: () => import('@/pages/MasterDataPage.vue'),
    meta: { roles: ['system_admin', 'data_steward', 'operations_analyst', 'standard_user'] },
  },
  {
    path: '/playback',
    name: 'Playback',
    component: () => import('@/pages/PlaybackPage.vue'),
    meta: { roles: ['system_admin', 'data_steward', 'operations_analyst', 'standard_user'] },
  },
  {
    path: '/analytics',
    name: 'Analytics',
    component: () => import('@/pages/AnalyticsPage.vue'),
    meta: { roles: ['operations_analyst', 'system_admin'] },
  },
  {
    path: '/ingestion',
    name: 'Ingestion',
    component: () => import('@/pages/IngestionPage.vue'),
    meta: { roles: ['system_admin', 'operations_analyst'] },
  },
  {
    path: '/reports',
    name: 'Reports',
    component: () => import('@/pages/ReportsPage.vue'),
    meta: { roles: ['system_admin', 'operations_analyst'] },
  },
  {
    path: '/security',
    name: 'SecurityAdmin',
    component: () => import('@/pages/SecurityAdminPage.vue'),
    meta: { roles: ['system_admin'] },
  },
  {
    path: '/',
    redirect: '/master/sku',
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/master/sku',
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, _from, next) => {
  if (to.meta.public) {
    return next();
  }

  const auth = useAuthStore();

  if (!auth.isAuthenticated) {
    return next({ path: '/login', query: { redirect: to.fullPath } });
  }

  if (to.meta.roles && to.meta.roles.length > 0) {
    if (!auth.hasAnyRole(to.meta.roles)) {
      // Find the first accessible route for this user as fallback
      const fallback = routes.find(
        (r) => !r.meta?.public && r.meta?.roles && auth.hasAnyRole(r.meta.roles) && r.path !== to.path
      );
      return next(fallback ? fallback.path.replace(':entity', 'sku') : '/master/sku');
    }
  }

  return next();
});

export default router;
