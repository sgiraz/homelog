<template>
  <div class="min-h-screen flex items-center justify-center bg-paper dark:bg-gray-900 p-4">
    <Card class="w-full max-w-md p-8">
      <div class="text-center mb-8">
        <h1 class="text-4xl font-bold mb-2 text-gray-900 dark:text-white">HomeLog</h1>
        <p class="text-gray-600 dark:text-gray-400">{{ t('auth.appTagline') }}</p>
      </div>

      <!-- ── Demo banner (only on demo instances) ── -->
      <div
        v-if="isDemoMode && mode === 'login'"
        class="mb-6 p-4 rounded-xl bg-paper-100 dark:bg-gray-800 border border-ember/20 dark:border-ember/30 space-y-3 text-center"
      >
        <p class="text-sm font-semibold text-ember-deep dark:text-ember-light">
          🎭 {{ t('demo.login.title') }}
        </p>
        <p class="text-xs text-gray-600 dark:text-gray-400">
          {{ t('demo.login.description') }}
        </p>
        <Button class="w-full" :disabled="loading" @click="enterDemo">
          {{ t('demo.login.button') }}
        </Button>
        <p class="text-xs text-gray-500 dark:text-gray-400 pt-1">
          {{ t('demo.login.selfHostPrompt') }}
          <a
            :href="links.github"
            target="_blank"
            rel="noopener"
            class="font-semibold underline hover:no-underline text-ember-deep dark:text-ember-light"
          >
            {{ t('demo.login.selfHostLink') }}
          </a>
        </p>
      </div>

      <!-- ── Login / Register ── -->
      <form v-if="mode === 'login' || mode === 'register'" @submit.prevent="handleSubmit" class="space-y-4">
        <Input
          v-model="form.email"
          :label="t('auth.email')"
          type="email"
          :placeholder="t('auth.emailPlaceholder')"
          required
          id="email"
          autocomplete="email"
        />

        <Input
          v-model="form.password"
          :label="t('auth.password')"
          type="password"
          :placeholder="t('auth.passwordPlaceholder')"
          required
          id="password"
          :autocomplete="mode === 'register' ? 'new-password' : 'current-password'"
        />

        <Input
          v-if="mode === 'register'"
          v-model="form.name"
          :label="t('auth.name')"
          :placeholder="t('auth.namePlaceholder')"
          required
          id="name"
          autocomplete="name"
        />

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <Button type="submit" class="w-full" :disabled="loading">
          {{ loading ? t('common.states.loading') : (mode === 'register' ? t('auth.register') : t('auth.signIn')) }}
        </Button>

        <div class="flex flex-col gap-2 items-center">
          <button
            type="button"
            @click="mode = mode === 'register' ? 'login' : 'register'; error = null"
            class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
          >
            {{ mode === 'register' ? t('auth.switchToLogin') : t('auth.switchToRegister') }}
          </button>

          <button
            v-if="mode === 'login'"
            type="button"
            @click="mode = 'forgot'; error = null; success = null"
            class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
          >
            {{ t('auth.forgotPasswordLink') }}
          </button>
        </div>
      </form>

      <!-- ── Forgot Password ── -->
      <div v-else-if="mode === 'forgot'" class="space-y-4">
        <div class="text-center mb-2">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('auth.forgot.title') }}</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {{ t('auth.forgot.subtitle') }}
          </p>
        </div>

        <Input
          v-model="forgotEmail"
          :label="t('auth.email')"
          type="email"
          :placeholder="t('auth.emailPlaceholder')"
          required
          id="forgot-email"
          autocomplete="email"
        />

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <!-- Token result box (dev mode: server returns the token inline) -->
        <div v-if="resetToken" class="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-xl space-y-3">
          <p class="text-sm font-medium text-amber-800 dark:text-amber-300">
            {{ t('auth.forgot.tokenGenerated') }}
          </p>
          <code class="block break-all text-xs bg-white dark:bg-gray-800 p-3 rounded-lg border border-amber-200 dark:border-amber-700 text-gray-900 dark:text-gray-100 select-all">
            {{ resetToken }}
          </code>
          <p class="text-xs text-amber-700 dark:text-amber-400">
            {{ t('auth.forgot.tokenInstructions') }}
          </p>
          <Button class="w-full" @click="mode = 'reset'; error = null">
            {{ t('auth.forgot.tokenUseButton') }}
          </Button>
        </div>

        <!-- Generic success (production: token is logged server-side, not returned) -->
        <div v-else-if="forgotSubmitted" class="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-xl space-y-3">
          <p class="text-sm text-blue-800 dark:text-blue-300">
            {{ t('auth.forgot.submittedMessage') }}
          </p>
          <Button class="w-full" @click="mode = 'reset'; error = null">
            {{ t('auth.forgot.submittedAction') }}
          </Button>
        </div>

        <Button v-else class="w-full" :disabled="loading" @click="handleForgotPassword">
          {{ loading ? t('auth.forgot.submitting') : t('auth.forgot.submit') }}
        </Button>

        <button
          type="button"
          @click="mode = 'login'; error = null; resetToken = null; forgotSubmitted = false"
          class="w-full text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400"
        >
          {{ t('auth.forgot.backToLogin') }}
        </button>
      </div>

      <!-- ── Reset Password ── -->
      <div v-else-if="mode === 'reset'" class="space-y-4">
        <div class="text-center mb-2">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('auth.reset.title') }}</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {{ t('auth.reset.subtitle') }}
          </p>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('auth.reset.tokenLabel') }}</label>
          <textarea
            v-model="resetForm.token"
            rows="2"
            :placeholder="t('auth.reset.tokenPlaceholder')"
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none font-mono"
          />
        </div>

        <Input
          v-model="resetForm.newPassword"
          :label="t('auth.reset.newPasswordLabel')"
          type="password"
          :placeholder="t('auth.reset.newPasswordPlaceholder')"
          id="new-password"
          autocomplete="new-password"
        />

        <Input
          v-model="resetForm.confirmPassword"
          :label="t('auth.reset.confirmPasswordLabel')"
          type="password"
          :placeholder="t('auth.reset.confirmPasswordPlaceholder')"
          id="confirm-password"
          autocomplete="new-password"
        />

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <div v-if="success" class="text-green-700 text-sm bg-green-50 dark:bg-green-900/20 p-3 rounded-lg">
          {{ success }}
        </div>

        <Button class="w-full" :disabled="loading" @click="handleResetPassword">
          {{ loading ? t('common.states.saving') : t('auth.reset.submit') }}
        </Button>

        <button
          type="button"
          @click="mode = 'forgot'; error = null; success = null"
          class="w-full text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400"
        >
          {{ t('auth.reset.backToForgot') }}
        </button>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { authAPI } from '@/api/client'
import { useDemoMode, DEMO_EMAIL, DEMO_PASSWORD } from '@/composables/useDemoMode'
import { LINKS as links } from '@/config/links'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const { isDemoMode } = useDemoMode()

// mode: 'login' | 'register' | 'forgot' | 'reset'
const mode = ref('login')
const loading = ref(false)
const error = ref(null)
const success = ref(null)

const form = ref({ email: '', password: '', name: '' })

const forgotEmail = ref('')
const resetToken = ref(null)
const forgotSubmitted = ref(false)

const resetForm = ref({ token: '', newPassword: '', confirmPassword: '' })

async function handleSubmit() {
  loading.value = true
  error.value = null
  try {
    if (mode.value === 'register') {
      await authStore.register(form.value)
    } else {
      await authStore.login(form.value)
    }
    router.push('/')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

// Prefill the shared demo credentials and sign in immediately.
async function enterDemo() {
  form.value.email = DEMO_EMAIL
  form.value.password = DEMO_PASSWORD
  await handleSubmit()
}

async function handleForgotPassword() {
  if (!forgotEmail.value) {
    error.value = t('auth.forgot.emailRequired')
    return
  }
  loading.value = true
  error.value = null
  resetToken.value = null
  forgotSubmitted.value = false
  try {
    const { data } = await authAPI.forgotPassword(forgotEmail.value)
    forgotSubmitted.value = true
    if (data.reset_token) {
      // Dev-mode backend returned the token inline
      resetToken.value = data.reset_token
      resetForm.value.token = data.reset_token
    }
  } catch (err) {
    error.value = err.response?.data?.error || t('auth.forgot.error')
  } finally {
    loading.value = false
  }
}

async function handleResetPassword() {
  error.value = null
  success.value = null

  if (!resetForm.value.token.trim()) {
    error.value = t('auth.reset.tokenRequired')
    return
  }
  if (resetForm.value.newPassword.length < 6) {
    error.value = t('auth.reset.passwordTooShort')
    return
  }
  if (resetForm.value.newPassword !== resetForm.value.confirmPassword) {
    error.value = t('auth.reset.passwordsMismatch')
    return
  }

  loading.value = true
  try {
    await authAPI.resetPassword(resetForm.value.token.trim(), resetForm.value.newPassword)
    success.value = t('auth.reset.success')
    setTimeout(() => {
      mode.value = 'login'
      error.value = null
      success.value = null
      resetToken.value = null
      forgotSubmitted.value = false
      resetForm.value = { token: '', newPassword: '', confirmPassword: '' }
    }, 2000)
  } catch (err) {
    error.value = err.response?.data?.error || t('auth.reset.error')
  } finally {
    loading.value = false
  }
}
</script>
