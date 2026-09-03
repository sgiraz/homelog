import { defineStore } from 'pinia'
import { ref } from 'vue'
import { expensesAPI } from '@/api/client'
import { apiErrorMessage } from '@/utils/apiError'

const PAGE_SIZE = 20

export const useExpensesStore = defineStore('expenses', () => {
  const expenses = ref([])
  const loading = ref(false)
  const saving = ref(false)
  const error = ref(null)
  const total = ref(0)
  const page = ref(1)
  const hasMore = ref(false)

  // Deduplication: track in-flight fetch
  let fetchPromise = null

  async function fetchExpenses(filters = {}, { page: p = 1, append = false } = {}) {
    // Deduplicate: only for non-append (initial load / filter change)
    if (!append && fetchPromise) return fetchPromise

    loading.value = true
    error.value = null

    const doFetch = async () => {
      try {
        const params = {
          ...filters,
          limit: PAGE_SIZE,
          offset: (p - 1) * PAGE_SIZE
        }
        const { data } = await expensesAPI.list(params)
        const fetched = data.expenses || []
        if (append) {
          // Dedup against any rows already present (e.g. an expense prepended
          // via ensureExpense for a deep-link target) so the user doesn't see
          // the same row twice when its natural page comes in.
          const seen = new Set(expenses.value.map(e => e.id))
          const deduped = fetched.filter(e => !seen.has(e.id))
          expenses.value = [...expenses.value, ...deduped]
        } else {
          expenses.value = fetched
        }
        total.value = data.total || 0
        hasMore.value = fetched.length === PAGE_SIZE
        page.value = p
      } catch (err) {
        error.value = apiErrorMessage(err)
        throw err
      } finally {
        loading.value = false
        if (!append) fetchPromise = null
      }
    }

    if (!append) {
      fetchPromise = doFetch()
      return fetchPromise
    }
    return doFetch()
  }

  async function fetchMore(filters = {}) {
    if (!hasMore.value || loading.value) return
    await fetchExpenses(filters, { page: page.value + 1, append: true })
  }

  // Fetch a single expense and insert it into the list if missing. Used by
  // deep-links from global search to surface rows that live past the first
  // page of the infinite paginator. No-ops if the id is already in the list.
  // `comparator` keeps the inserted row in the same sort order the user sees
  // — without it an old expense would land at the top, which is confusing.
  async function ensureExpense(id, { comparator } = {}) {
    if (!id) return null
    const numericId = Number(id)
    const existing = expenses.value.find(e => e.id === numericId)
    if (existing) return existing
    try {
      const { data } = await expensesAPI.get(numericId)
      if (data && !expenses.value.some(e => e.id === data.id)) {
        if (typeof comparator === 'function') {
          const list = [...expenses.value, data]
          list.sort(comparator)
          expenses.value = list
        } else {
          expenses.value = [data, ...expenses.value]
        }
      }
      return data
    } catch (err) {
      error.value = apiErrorMessage(err)
      return null
    }
  }

  async function createExpense(expense) {
    saving.value = true
    error.value = null
    try {
      const { data } = await expensesAPI.create(expense)
      expenses.value.unshift(data)
      total.value++
      return data
    } catch (err) {
      error.value = apiErrorMessage(err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function updateExpense(id, expense) {
    saving.value = true
    error.value = null
    try {
      const { data } = await expensesAPI.update(id, expense)
      const index = expenses.value.findIndex(e => e.id === id)
      if (index !== -1) {
        expenses.value[index] = data
      }
      return data
    } catch (err) {
      error.value = apiErrorMessage(err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function deleteExpense(id) {
    saving.value = true
    error.value = null
    try {
      await expensesAPI.delete(id)
      expenses.value = expenses.value.filter(e => e.id !== id)
      total.value--
    } catch (err) {
      error.value = apiErrorMessage(err)
      throw err
    } finally {
      saving.value = false
    }
  }

  return {
    expenses, loading, saving, error, total, page, hasMore,
    fetchExpenses, fetchMore, ensureExpense, createExpense, updateExpense, deleteExpense
  }
})
