import { defineStore } from 'pinia'
import { ref } from 'vue'
import { expensesAPI } from '@/api/client'

export const useExpensesStore = defineStore('expenses', () => {
  const expenses = ref([])
  const loading = ref(false)
  const error = ref(null)
  const total = ref(0)

  async function fetchExpenses(filters = {}) {
    loading.value = true
    error.value = null
    try {
      const { data } = await expensesAPI.list(filters)
      expenses.value = data.expenses || []
      total.value = data.total || 0
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    } finally {
      loading.value = false
    }
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
      // Replace the expense in the list
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

  return { expenses, loading, error, total, fetchExpenses, createExpense, updateExpense, deleteExpense }
})
