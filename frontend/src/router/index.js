import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue')
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/expenses',
      name: 'expenses',
      component: () => import('../views/ExpensesView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/balance',
      redirect: '/expenses?tab=bilancio'
    },
    {
      path: '/utilities',
      name: 'utilities',
      component: () => import('../views/UtilitiesView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/utilities/:id',
      name: 'utility-detail',
      component: () => import('../views/UtilityDetailView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/projects',
      name: 'projects',
      component: () => import('../views/ProjectsView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/notifications',
      name: 'notifications',
      component: () => import('../views/NotificationsView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/SettingsView.vue'),
      meta: { requiresAuth: true }
    }
  ]
})

// Navigation guard for authentication
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/')
  } else {
    next()
  }
})

export default router
