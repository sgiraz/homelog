<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
    <Card class="w-full max-w-md p-8">
      <div class="text-center mb-8">
        <h1 class="text-4xl font-bold mb-2 text-gray-900 dark:text-white">HomeLog</h1>
        <p class="text-gray-600 dark:text-gray-400">Gestione Spese Domestiche</p>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
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
          :autocomplete="isRegister ? 'new-password' : 'current-password'"
        />

        <Input
          v-if="isRegister"
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
          {{ loading ? 'Caricamento...' : (isRegister ? 'Registrati' : 'Accedi') }}
        </Button>

        <button
          type="button"
          @click="toggleMode"
          class="w-full text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
        >
          {{ isRegister ? 'Hai gia un account? Accedi' : 'Non hai un account? Registrati' }}
        </button>
      </form>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/common/Card.vue'
import Input from '@/components/common/Input.vue'
import Button from '@/components/common/Button.vue'

const router = useRouter()
const authStore = useAuthStore()

const isRegister = ref(false)
const loading = ref(false)
const error = ref(null)

const form = ref({
  email: '',
  password: '',
  name: ''
})

function toggleMode() {
  isRegister.value = !isRegister.value
  error.value = null
}

async function handleSubmit() {
  loading.value = true
  error.value = null

  try {
    if (isRegister.value) {
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
</script>
