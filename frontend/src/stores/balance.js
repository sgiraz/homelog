import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { balanceAPI, expensesAPI, settlementsAPI } from '@/api/client'
import { apiErrorMessage } from '@/utils/apiError'

export const useBalanceStore = defineStore('balance', () => {
  const balance = ref(0)
  const currentMemberName = ref('')
  const currentMemberId = ref(null)
  const otherMemberName = ref('')
  const otherMemberId = ref(null)
  // Every other member with their own balance, the shown one first. With more
  // than one, the user picks who to look at; `selected` remembers that pick
  // for the household it was made in, null meaning "the server's choice".
  const counterparts = ref([])
  const selected = ref({ householdId: null, memberId: null })
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

  async function fetchBalanceDetails(householdId) {
    loading.value = true
    error.value = null
    try {
      const { data } = await withCounterpart(householdId, (memberId) => balanceAPI.details(householdId, memberId))
      balance.value = data.balance || 0
      currentMemberName.value = data.current_member_name || ''
      currentMemberId.value = data.current_member_id || null
      otherMemberName.value = data.other_member_name || ''
      otherMemberId.value = data.other_member_id || null
      counterparts.value = data.counterparts || []
      unsettledSplits.value = data.unsettled_splits || []
      settlements.value = data.settlements || []
    } catch (err) {
      balance.value = 0
      counterparts.value = []
      unsettledSplits.value = []
      settlements.value = []
      error.value = apiErrorMessage(err)
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
      const householdId = settlementData.household_id
      await fetchBalanceDetails(householdId)
      if (settlementData.target_expense_id) await fetchDebts(householdId)
      return data
    } catch (err) {
      error.value = apiErrorMessage(err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function fetchDebts(householdId) {
    debtsLoading.value = true
    error.value = null
    try {
      const { data } = await withCounterpart(householdId, (memberId) => balanceAPI.debts(householdId, memberId))
      debts.value = data.debts || []
      totalIOwe.value = data.total_i_owe || 0
      totalTheyOwe.value = data.total_they_owe || 0
      if (data.current_member_id) currentMemberId.value = data.current_member_id
      if (data.current_member_name) currentMemberName.value = data.current_member_name
      if (data.other_member_id) otherMemberId.value = data.other_member_id
      if (data.other_member_name) otherMemberName.value = data.other_member_name
      if (data.counterparts) counterparts.value = data.counterparts
    } catch (err) {
      debts.value = []
      totalIOwe.value = 0
      totalTheyOwe.value = 0
      error.value = apiErrorMessage(err)
    } finally {
      debtsLoading.value = false
    }
  }

  function selectedFor(householdId) {
    return selected.value.householdId === householdId ? selected.value.memberId : null
  }

  // Runs a request against the picked counterpart, or the server's choice
  // without one. A pick can vanish (an admin deletes that virtual member):
  // then it is forgotten and the server chooses again, instead of the section
  // failing until a reload.
  async function withCounterpart(householdId, request) {
    const memberId = selectedFor(householdId)
    try {
      return await request(memberId)
    } catch (err) {
      const code = err?.response?.data?.error_code
      if (memberId == null || (code !== 'other_member_not_found' && code !== 'invalid_other_member_id')) throw err
      if (selectedFor(householdId) === memberId) selected.value = { householdId: null, memberId: null }
      return request(null)
    }
  }

  async function selectCounterpart(householdId, memberId) {
    selected.value = { householdId, memberId }
    await Promise.all([fetchBalanceDetails(householdId), fetchDebts(householdId)])
  }

  // Moves an expense in or out of the long-term debt ledger. Both ledgers are
  // refreshed: the share leaves one list and appears in the other.
  async function setLongTermDebt(expenseId, isLongTermDebt, householdId) {
    saving.value = true
    error.value = null
    try {
      const { data } = await expensesAPI.setLongTermDebt(expenseId, isLongTermDebt)
      await Promise.all([fetchBalanceDetails(householdId), fetchDebts(householdId)])
      return data
    } catch (err) {
      error.value = apiErrorMessage(err)
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
      const householdId = payload.household_id
      await Promise.all([fetchBalanceDetails(householdId), fetchDebts(householdId)])
      return data
    } catch (err) {
      error.value = apiErrorMessage(err)
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
    counterparts,
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
    fetchBalanceDetails,
    createSettlement,
    fetchDebts,
    selectCounterpart,
    setLongTermDebt,
    createCompensation
  }
})
