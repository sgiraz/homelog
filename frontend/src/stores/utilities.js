import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { utilitiesAPI } from '@/api/client'

export const useUtilitiesStore = defineStore('utilities', () => {
  const utilities = ref([])
  const loading = ref(false)
  const error = ref(null)
  const selectedUtility = ref(null)

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
    loading.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.list(params)
      utilities.value = data || []
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to fetch utilities'
      console.error('Error fetching utilities:', err)
    } finally {
      loading.value = false
    }
  }

  async function fetchUtility(id) {
    loading.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.get(id)
      selectedUtility.value = data
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to fetch utility'
      console.error('Error fetching utility:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createUtility(utilityData) {
    loading.value = true
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
      loading.value = false
    }
  }

  async function updateUtility(id, utilityData) {
    loading.value = true
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
      loading.value = false
    }
  }

  async function deleteUtility(id) {
    loading.value = true
    error.value = null

    try {
      await utilitiesAPI.delete(id)
      utilities.value = utilities.value.filter(u => u.id !== id)
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to delete utility'
      console.error('Error deleting utility:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Bill actions
  async function addBill(utilityId, billData) {
    loading.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.addBill(utilityId, billData)
      // Refresh the utility to get updated bills
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to add bill'
      console.error('Error adding bill:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateBill(utilityId, billId, billData) {
    loading.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.updateBill(utilityId, billId, billData)
      // Refresh the utility to get updated bills
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update bill'
      console.error('Error updating bill:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateBillFull(utilityId, billId, billData) {
    loading.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.updateBillFull(utilityId, billId, billData)
      // Refresh the utility to get updated bills
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update bill'
      console.error('Error updating bill:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteBill(utilityId, billId) {
    loading.value = true
    error.value = null

    try {
      await utilitiesAPI.deleteBill(utilityId, billId)
      // Refresh the utility to get updated bills
      await fetchUtility(utilityId)
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to delete bill'
      console.error('Error deleting bill:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Reading actions
  async function addReading(utilityId, readingData) {
    loading.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.addReading(utilityId, readingData)
      // Refresh the utility to get updated readings
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to add reading'
      console.error('Error adding reading:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateReading(utilityId, readingId, readingData) {
    loading.value = true
    error.value = null

    try {
      const { data } = await utilitiesAPI.updateReading(utilityId, readingId, readingData)
      // Refresh the utility to get updated readings
      await fetchUtility(utilityId)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update reading'
      console.error('Error updating reading:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteReading(utilityId, readingId) {
    loading.value = true
    error.value = null

    try {
      await utilitiesAPI.deleteReading(utilityId, readingId)
      // Refresh the utility to get updated readings
      await fetchUtility(utilityId)
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to delete reading'
      console.error('Error deleting reading:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  function clearSelectedUtility() {
    selectedUtility.value = null
  }

  return {
    // State
    utilities,
    loading,
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
