import { utilitiesAPI } from '@/api/client'

/**
 * Opens a bill's PDF in a new tab.
 *
 * PDFs are private and a plain <a href> cannot send the Authorization header, so
 * the file is fetched through the API client and shown from an object URL. The
 * tab is opened synchronously, before the request, so it is not treated as a
 * popup. Rejects with an axios-shaped error and closes the tab on failure.
 */
export async function openBillPdf(utilityId, billId) {
  const tab = window.open('', '_blank')
  try {
    const { data } = await utilitiesAPI.getBillPDF(utilityId, billId)
    const url = URL.createObjectURL(new Blob([data], { type: 'application/pdf' }))
    if (tab) tab.location.href = url
    else window.open(url, '_blank')
    // The tab keeps its own reference once navigated; free ours later.
    setTimeout(() => URL.revokeObjectURL(url), 60_000)
  } catch (err) {
    tab?.close()
    // responseType 'blob' also wraps the JSON error body in a Blob.
    const body = err?.response?.data
    if (body instanceof Blob) {
      try {
        err.response.data = JSON.parse(await body.text())
      } catch {
        err.response.data = null
      }
    }
    throw err
  }
}
