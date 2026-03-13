import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { utilitiesAPI } from '@/api/client'

export const useUtilitiesStore = defineStore('utilities', () => {
  const utilities = ref([])
  const loading = ref(false)
  const saving = ref(false)
  const error = ref(null)
  const selectedUtility = ref(null)

  // Deduplication: track in-flight fetch to prevent parallel calls
  let fetchListPromise = null
  let fetchDetailPromise = null

  // Computed properties
  const unpaidBillsCount = computed(() => {
    return utilities.value.reduce((count, utility) => {
      const unpaid = utility.bills?.filter(b => !b.is_paid) || []
      return count + unpaid.length
    }, 0)
  })

  const dueSoonBillsCount = computed(() => {
    const now = new Date()
    const threeDaysFromNow = new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000)

    return utilities.value.reduce((count, utility) => {
      const dueSoon = utility.bills?.filter(b => {
        if (b.is_paid) return false
        const dueDate = new Date(b.due_date)
        return dueDate <= threeDaysFromNow && dueDate >= now
      }) || []
      return count + dueSoon.length
    }, 0)
  })

  const utilitiesByType = computed(() => {
    const grouped = {
      electricity: [],
      gas: [],
      water: [],
      waste: []
    }

    utilities.value.forEach(utility => {
      if (grouped[utility.type]) {
        grouped[utility.type].push(utility)
      }
    })

    return grouped
  })

  // Actions
  async function fetchUtilities(params = {}) {
    // Deduplicate: if already fetching with same params, return existing promise
    if (fetchListPromise) return fetchListPromise

    loading.value = true
    error.value = null

    fetchListPromise = utilitiesAPI.list(params)
      .then(({ data }) => {
        utilities.value = data || []
      })
      .catch(err => {
        error.value = err.response?.data?.error || 'Failed to fetch utilities'
        console.error('Error fetching utilities:', err)
      })
      .finally(() => {
        loading.value = false
        fetchListPromise = null
      })

    return fetchListPromise
  }

  async function fetchUtility(id) {
    // Deduplicate: if already fetching this utility, return existing promise
    if (fetchDetailPromise) return fetchDetailPromise

    error.value = null

    fetchDetailPromise = utilitiesAPI.get(id)
      .then(({ data }) => {
        selectedUtility.value = data
        return data
      })
      .catch(err => {
        error.value = err.response?.data?.error || 'Failed to fetch utility'
        console.error('Error fetching utility:', err)
        throw err
      })
      .finally(() => {
        fetchDetailPromise = null
      })

    return fetchDetailPromise
  }

  async function createUtility(utilityData) {
    saving.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.create(utilityData)
      utilities.value.push(data)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to create utility'
      console.error('Error creating utility:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function updateUtility(id, utilityData) {
    saving.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.update(id, utilityData)
      const index = utilities.value.findIndex(u => u.id === id)
      if (index !== -1) {
        utilities.value[index] = data
      }
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update utility'
      console.error('Error updating utility:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function deleteUtility(id) {
    saving.value = true
    error.value = null

    try {
      await utilitiesAPI.delete(id)
      utilities.value = utilities.value.filter(u => u.id !== id)
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to delete utility'
      console.error('Error deleting utility:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  // Bill actions — use saving flag, fetchUtility without touching loading
  async function addBill(utilityId, billData) {
    saving.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.addBill(utilityId, billData)
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to add bill'
      console.error('Error adding bill:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function updateBill(utilityId, billId, billData) {
    saving.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.updateBill(utilityId, billId, billData)
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update bill'
      console.error('Error updating bill:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function updateBillFull(utilityId, billId, billData) {
    saving.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.updateBillFull(utilityId, billId, billData)
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update bill'
      console.error('Error updating bill:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function deleteBill(utilityId, billId) {
    saving.value = true
    error.value = null

    try {
      await utilitiesAPI.deleteBill(utilityId, billId)
      await fetchUtility(utilityId)
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to delete bill'
      console.error('Error deleting bill:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  // Reading actions
  async function addReading(utilityId, readingData) {
    saving.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.addReading(utilityId, readingData)
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to add reading'
      console.error('Error adding reading:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function updateReading(utilityId, readingId, readingData) {
    saving.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.updateReading(utilityId, readingId, readingData)
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update reading'
      console.error('Error updating reading:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function deleteReading(utilityId, readingId) {
    saving.value = true
    error.value = null

    try {
      await utilitiesAPI.deleteReading(utilityId, readingId)
      await fetchUtility(utilityId)
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to delete reading'
      console.error('Error deleting reading:', err)
      throw err
    } finally {
      saving.value = false
    }
  }

  function clearSelectedUtility() {
    selectedUtility.value = null
  }

  return {
    // State
    utilities,
    loading,
    saving,
    error,
    selectedUtility,

    // Computed
    unpaidBillsCount,
    dueSoonBillsCount,
    utilitiesByType,

    // Actions
    fetchUtilities,
    fetchUtility,
    createUtility,
    updateUtility,
    deleteUtility,
    addBill,
    updateBill,
    updateBillFull,
    deleteBill,
    addReading,
    updateReading,
    deleteReading,
    clearSelectedUtility
  }
})
