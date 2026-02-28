import { defineStore } from 'pinia'
import { ref } from 'vue'
import { expensesAPI } from '@/api/client'

const PAGE_SIZE = 20

export const useExpensesStore = defineStore('expenses', () => {
  const expenses = ref([])
  const loading = ref(false)
  const error = ref(null)
  const total = ref(0)
  const page = ref(1)
  const hasMore = ref(false)

  async function fetchExpenses(filters = {}, { page: p = 1, append = false } = {}) {
    loading.value = true
    error.value = null
    try {
      const params = {
        ...filters,
        limit: PAGE_SIZE,
        offset: (p - 1) * PAGE_SIZE
      }
      const { data } = await expensesAPI.list(params)
      const fetched = data.expenses || []
      if (append) {
        expenses.value = [...expenses.value, ...fetched]
      } else {
        expenses.value = fetched
      }
      total.value = data.total || 0
      hasMore.value = fetched.length === PAGE_SIZE
      page.value = p
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchMore(filters = {}) {
    if (!hasMore.value || loading.value) return
    await fetchExpenses(filters, { page: page.value + 1, append: true })
  }

  async function createExpense(expense) {
    try {
      const { data } = await expensesAPI.create(expense)
      expenses.value.unshift(data)
      total.value++
      return data
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    }
  }

  async function updateExpense(id, expense) {
    try {
      const { data } = await expensesAPI.update(id, expense)
      const index = expenses.value.findIndex(e => e.id === id)
      if (index !== -1) {
        expenses.value[index] = data
      }
      return data
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    }
  }

  async function deleteExpense(id) {
    try {
      await expensesAPI.delete(id)
      expenses.value = expenses.value.filter(e => e.id !== id)
      total.value--
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    }
  }

  return {
    expenses, loading, error, total, page, hasMore,
    fetchExpenses, fetchMore, createExpense, updateExpense, deleteExpense
  }
})
