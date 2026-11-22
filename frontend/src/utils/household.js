import { propertiesAPI } from '@/api/client'

// Members, balance and split settings belong to a household; a home only
// knows which household it is in. Resolves that household for a home id.
export async function householdIdOfProperty(propertyId) {
  if (!propertyId) return null
  const { data } = await propertiesAPI.list()
  return (data || []).find(p => p.id === propertyId)?.household_id ?? null
}
