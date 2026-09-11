import { utilityTypeStyle } from '@/config/utilityTypes'

/**
 * Icon and tile colour for a notification row.
 *
 * The navbar dropdown and the notifications page each kept an identical copy
 * of this mapping. Service notifications take the service type's identity
 * from config/utilityTypes; the rest are keyed by notification kind.
 */

// In-app notifications that are not about a service.
const NOTIFICATION_KINDS = {
  join_request: { icon: '👤', tile: 'bg-violet-100 dark:bg-violet-900/30' },
  expense_shared: { icon: '💳', tile: 'bg-emerald-100 dark:bg-emerald-900/30' }
}
const GENERIC = { icon: '🔔', tile: 'bg-surface-2' }
// A service notification whose service no longer exists still needs an icon.
const NO_SERVICE = { icon: '📬', tile: 'bg-surface-2' }

function styleOf(notif) {
  if (notif._source === 'notification') return NOTIFICATION_KINDS[notif.type] || GENERIC
  const type = notif.utility?.type
  return type ? utilityTypeStyle(type) : NO_SERVICE
}

export const notificationIcon = (notif) => styleOf(notif).icon
export const notificationTile = (notif) => styleOf(notif).tile
