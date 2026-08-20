import { createRouter, createWebHistory } from 'vue-router'
import { useSettingsStore } from '../stores/settings'

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
      path: '/projects/:id',
      name: 'project-detail',
      component: () => import('../views/ProjectDetailView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/notifications',
      name: 'notifications',
      component: () => import('../views/NotificationsView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/search',
      name: 'search',
      component: () => import('../views/SearchView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/SettingsView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/onboarding',
      name: 'onboarding',
      component: () => import('../views/OnboardingView.vue'),
      meta: { requiresAuth: true }
    }
  ]
})

// Navigation guard for authentication and onboarding
router.beforeEach(async (to, from, next) => {
  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth && !token) {
    next('/login')
    return
  }

  if (to.path === '/login' && token) {
    next('/')
    return
  }

  // Onboarding check: only for authenticated routes, not for /onboarding itself
  if (token && to.path !== '/onboarding' && to.meta.requiresAuth) {
    const settingsStore = useSettingsStore()
    // Load settings if not yet loaded (e.g. first navigation after login)
    if (!settingsStore.loaded) {
      try {
        await settingsStore.loadSettings()
      } catch {
        // If settings fail to load, don't block navigation
      }
    }
    // hasProperty vetoes the redirect: someone already in a household has
    // nothing left to onboard, and the wizard could only offer to create or
    // join a property they are already a member of. Accounts predating the
    // onboarding flag (no user_settings row -> the API answers with
    // onboarding_completed=false) would otherwise be trapped here forever.
    if (settingsStore.loaded && !settingsStore.onboardingCompleted && !settingsStore.hasProperty) {
      next('/onboarding')
      return
    }
  }

  next()
})

export default router
