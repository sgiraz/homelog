import { reactive } from 'vue'

// confirmText/cancelText default to '' so ConfirmDialog can render translated
// fallbacks (common.actions.confirm/cancel) via i18n. Callers can still pass
// custom strings for context-specific wording.
const state = reactive({
  show: false,
  title: '',
  message: '',
  confirmText: '',
  cancelText: '',
  variant: 'default',
  resolve: null
})

export function useConfirm() {
  function confirm({ title = '', message = '', confirmText = '', cancelText = '', variant = 'default' } = {}) {
    return new Promise((resolve) => {
      state.show = true
      state.title = title
      state.message = message
      state.confirmText = confirmText
      state.cancelText = cancelText
      state.variant = variant
      state.resolve = resolve
    })
  }

  function handleConfirm() {
    state.resolve?.(true)
    state.show = false
  }

  function handleCancel() {
    state.resolve?.(false)
    state.show = false
  }

  return { state, confirm, handleConfirm, handleCancel }
}
