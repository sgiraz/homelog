import { defineStore } from 'pinia'
import { ref } from 'vue'
import { balanceAPI, settlementsAPI } from '@/api/client'

export const useBalanceStore = defineStore('balance', () => {
  const balance = ref(0)
  const currentMemberName = ref('')
  const currentMemberId = ref(null)
  const otherMemberName = ref('')
  const otherMemberId = ref(null)
  const message = ref('')
  const unsettledSplits = ref([])
  const settlements = ref([])
  const loading = ref(false)
  const saving = ref(false)
  const error = ref(null)

  async function fetchBalance(propertyId = 1) {
    loading.value = true
    error.value = null
    try {
      const { data } = await balanceAPI.get(propertyId)
      balance.value = data.balance || 0
      currentMemberName.value = data.current_member_name || ''
      currentMemberId.value = data.current_member_id || null
      otherMemberName.value = data.other_member_name || ''
      otherMemberId.value = data.other_member_id || null
      message.value = data.message || ''
    } catch (err) {
      balance.value = 0
      error.value = err.response?.data?.error || err.message
    } finally {
      loading.value = false
    }
  }

  async function fetchBalanceDetails(propertyId = 1) {
    loading.value = true
    error.value = null
    try {
      const { data } = await balanceAPI.details(propertyId)
      balance.value = data.balance || 0
      currentMemberName.value = data.current_member_name || ''
      currentMemberId.value = data.current_member_id || null
      otherMemberName.value = data.other_member_name || ''
      otherMemberId.value = data.other_member_id || null
      unsettledSplits.value = data.unsettled_splits || []
      settlements.value = data.settlements || []
    } catch (err) {
      balance.value = 0
      unsettledSplits.value = []
      settlements.value = []
      error.value = err.response?.data?.error || err.message
    } finally {
      loading.value = false
    }
  }

  async function createSettlement(settlementData) {
    saving.value = true
    error.value = null
    try {
      const { data } = await settlementsAPI.create(settlementData)
      // Refresh balance after settlement
      await fetchBalanceDetails(settlementData.property_id || 1)
      return data
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    } finally {
      saving.value = false
    }
  }

  async function fetchSettlements(propertyId = 1) {
    try {
      const { data } = await settlementsAPI.list(propertyId)
      settlements.value = data.settlements || []
    } catch (err) {
      settlements.value = []
      error.value = err.response?.data?.error || err.message
    }
  }

  return {
    balance,
    currentMemberName,
    currentMemberId,
    otherMemberName,
    otherMemberId,
    message,
    unsettledSplits,
    settlements,
    loading,
    saving,
    error,
    fetchBalance,
    fetchBalanceDetails,
    createSettlement,
    fetchSettlements
  }
})
