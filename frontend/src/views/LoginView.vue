<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
    <Card class="w-full max-w-md p-8">
      <div class="text-center mb-8">
        <h1 class="text-4xl font-bold mb-2 text-gray-900 dark:text-white">HomeLog</h1>
        <p class="text-gray-600 dark:text-gray-400">Gestione Spese Domestiche</p>
      </div>

      <!-- ── Login / Register ── -->
      <form v-if="mode === 'login' || mode === 'register'" @submit.prevent="handleSubmit" class="space-y-4">
        <Input
          v-model="form.email"
          label="Email"
          type="email"
          placeholder="email@example.com"
          required
          id="email"
          autocomplete="email"
        />

        <Input
          v-model="form.password"
          label="Password"
          type="password"
          placeholder="Password"
          required
          id="password"
          :autocomplete="mode === 'register' ? 'new-password' : 'current-password'"
        />

        <Input
          v-if="mode === 'register'"
          v-model="form.name"
          label="Nome"
          placeholder="Il tuo nome"
          required
          id="name"
          autocomplete="name"
        />

        <div v-if="error" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
          {{ error }}
        </div>

        <Button type="submit" class="w-full" :disabled="loading">
          {{ loading ? 'Caricamento...' : (mode === 'register' ? 'Registrati' : 'Accedi') }}
        </Button>

        <div class="flex flex-col gap-2 items-center">
          <button
            type="button"
            @click="mode = mode === 'register' ? 'login' : 'register'; error = null"
            class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
          >
            {{ mode === 'register' ? 'Hai già un account? Accedi' : 'Non hai un account? Registrati' }}
          </button>

          <button
            v-if="mode === 'login'"
            type="button"
            @click="mode = 'forgot'; error = null; success = null"
            class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
          >
            Password dimenticata?
          </button>
        </div>
      </form>

      <!-- ── Forgot Password ── -->
      <div v-else-if="mode === 'forgot'" class="space-y-4">
        <div class="text-center mb-2">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">Recupero password</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Inserisci la tua email per generare un token di reset.
          </p>
        </div>

        <Input
          v-model="forgotEmail"
          label="Email"
          type="email"
          placeholder="email@example.com"
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
            Token generato (valido 1 ora):
          </p>
          <code class="block break-all text-xs bg-white dark:bg-gray-800 p-3 rounded-lg border border-amber-200 dark:border-amber-700 text-gray-900 dark:text-gray-100 select-all">
            {{ resetToken }}
          </code>
          <p class="text-xs text-amber-700 dark:text-amber-400">
            Copia questo token e usalo nel form di reset password qui sotto.
          </p>
          <Button class="w-full" @click="mode = 'reset'; error = null">
            Usa il token per reimpostare la password
          </Button>
        </div>

        <!-- Generic success (production: token is logged server-side, not returned) -->
        <div v-else-if="forgotSubmitted" class="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-xl space-y-3">
          <p class="text-sm text-blue-800 dark:text-blue-300">
            Se l'email è registrata, è stato generato un token di reset (valido 1 ora).
            Chiedi all'amministratore di recuperare il token dai log del server, poi procedi al reset.
          </p>
          <Button class="w-full" @click="mode = 'reset'; error = null">
            Ho il token, procedi al reset
          </Button>
        </div>

        <Button v-else class="w-full" :disabled="loading" @click="handleForgotPassword">
          {{ loading ? 'Invio...' : 'Genera token di reset' }}
        </Button>

        <button
          type="button"
          @click="mode = 'login'; error = null; resetToken = null; forgotSubmitted = false"
          class="w-full text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400"
        >
          ← Torna al login
        </button>
      </div>

      <!-- ── Reset Password ── -->
      <div v-else-if="mode === 'reset'" class="space-y-4">
        <div class="text-center mb-2">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">Nuova password</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Incolla il token ricevuto e scegli la nuova password.
          </p>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Token di reset</label>
          <textarea
            v-model="resetForm.token"
            rows="2"
            placeholder="Incolla qui il token..."
            class="w-full px-3 py-3 border border-gray-200 dark:border-gray-700 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none font-mono"
          />
        </div>

        <Input
          v-model="resetForm.newPassword"
          label="Nuova password"
          type="password"
          placeholder="Minimo 6 caratteri"
          id="new-password"
          autocomplete="new-password"
        />

        <Input
          v-model="resetForm.confirmPassword"
          label="Conferma password"
          type="password"
          placeholder="Ripeti la nuova password"
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
          {{ loading ? 'Salvataggio...' : 'Reimposta password' }}
        </Button>

        <button
          type="button"
          @click="mode = 'forgot'; error = null; success = null"
          class="w-full text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400"
        >
          ← Torna al recupero password
        </button>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const router = useRouter()
const authStore = useAuthStore()

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

async function handleForgotPassword() {
  if (!forgotEmail.value) {
    error.value = 'Inserisci la tua email.'
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
    error.value = err.response?.data?.error || 'Errore durante la generazione del token.'
  } finally {
    loading.value = false
  }
}

async function handleResetPassword() {
  error.value = null
  success.value = null

  if (!resetForm.value.token.trim()) {
    error.value = 'Il token è obbligatorio.'
    return
  }
  if (resetForm.value.newPassword.length < 6) {
    error.value = 'La nuova password deve avere almeno 6 caratteri.'
    return
  }
  if (resetForm.value.newPassword !== resetForm.value.confirmPassword) {
    error.value = 'Le due password non coincidono.'
    return
  }

  loading.value = true
  try {
    await authAPI.resetPassword(resetForm.value.token.trim(), resetForm.value.newPassword)
    success.value = 'Password reimpostata con successo! Puoi ora accedere.'
    setTimeout(() => {
      mode.value = 'login'
      error.value = null
      success.value = null
      resetToken.value = null
      forgotSubmitted.value = false
      resetForm.value = { token: '', newPassword: '', confirmPassword: '' }
    }, 2000)
  } catch (err) {
    error.value = err.response?.data?.error || 'Token non valido o scaduto.'
  } finally {
    loading.value = false
  }
}
</script>
