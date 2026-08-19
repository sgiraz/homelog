import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { balanceAPI, expensesAPI, settlementsAPI } from '@/api/client'

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

  // Long-term debts: big one-off imbalances kept out of `balance` on purpose,
  // repaid on their own schedule from the Debiti tab.
  const debts = ref([])
  const debtsLoading = ref(false)
  const totalIOwe = ref(0)
  const totalTheyOwe = ref(0)

  const openDebts = computed(() => debts.value.filter(d => !d.is_fully_repaid))
  const myOpenDebts = computed(() => openDebts.value.filter(d => d.i_owe))

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
      // Refresh balance after settlement — plus the debt ledger when the
      // payment was aimed at one of them.
      const propertyId = settlementData.property_id || 1
      await fetchBalanceDetails(propertyId)
      if (settlementData.target_expense_id) await fetchDebts(propertyId)
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

  async function fetchDebts(propertyId = 1, otherId = null) {
    debtsLoading.value = true
    error.value = null
    try {
      const { data } = await balanceAPI.debts(propertyId, otherId ?? otherMemberId.value)
      debts.value = data.debts || []
      totalIOwe.value = data.total_i_owe || 0
      totalTheyOwe.value = data.total_they_owe || 0
      if (data.current_member_id) currentMemberId.value = data.current_member_id
      if (data.current_member_name) currentMemberName.value = data.current_member_name
      if (data.other_member_id) otherMemberId.value = data.other_member_id
      if (data.other_member_name) otherMemberName.value = data.other_member_name
    } catch (err) {
      debts.value = []
      totalIOwe.value = 0
      totalTheyOwe.value = 0
      error.value = err.response?.data?.error || err.message
    } finally {
      debtsLoading.value = false
    }
  }

  // Moves an expense in or out of the long-term debt ledger. Both ledgers are
  // refreshed: the share leaves one list and appears in the other.
  async function setLongTermDebt(expenseId, isLongTermDebt, propertyId = 1) {
    saving.value = true
    error.value = null
    try {
      const { data } = await expensesAPI.setLongTermDebt(expenseId, isLongTermDebt)
      await Promise.all([fetchBalanceDetails(propertyId), fetchDebts(propertyId)])
      return data
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    } finally {
      saving.value = false
    }
  }

  async function createCompensation(payload) {
    saving.value = true
    error.value = null
    try {
      const { data } = await settlementsAPI.compensate(payload)
      const propertyId = payload.property_id || 1
      await Promise.all([fetchBalanceDetails(propertyId), fetchDebts(propertyId)])
      return data
    } catch (err) {
      error.value = err.response?.data?.error || err.message
      throw err
    } finally {
      saving.value = false
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
    debts,
    debtsLoading,
    totalIOwe,
    totalTheyOwe,
    openDebts,
    myOpenDebts,
    fetchBalance,
    fetchBalanceDetails,
    createSettlement,
    fetchSettlements,
    fetchDebts,
    setLongTermDebt,
    createCompensation
  }
})
