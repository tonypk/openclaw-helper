import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'welcome',
      component: () => import('../views/WelcomeView.vue'),
    },
    {
      path: '/check',
      name: 'check',
      component: () => import('../views/SystemCheckView.vue'),
    },
    {
      path: '/install',
      name: 'install',
      component: () => import('../views/InstallView.vue'),
    },
    {
      path: '/config',
      name: 'config',
      component: () => import('../views/ConfigView.vue'),
    },
    {
      path: '/success',
      name: 'success',
      meta: { requiresInstall: true },
      component: () => import('../views/SuccessView.vue'),
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      meta: { requiresInstall: true },
      component: () => import('../views/DashboardView.vue'),
    },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresInstall) {
    const installed = localStorage.getItem('openclaw_installed') === 'true'
    if (!installed) {
      return { name: 'welcome' }
    }
  }
})

export default router
