import axios from 'axios'

const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' }
})

// JWT interceptor
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// Refresh token interceptor
let isRefreshing = false
let failedQueue = []

function processQueue(error, token = null) {
  failedQueue.forEach(({ resolve, reject }) => {
    if (error) reject(error)
    else resolve(token)
  })
  failedQueue = []
}

apiClient.interceptors.response.use(
  response => response,
  async error => {
    const originalRequest = error.config
    // Skip refresh for the refresh endpoint itself to avoid infinite loop
    const isRefreshRequest = originalRequest.url?.includes('/auth/refresh')
    if (error.response?.status === 401 && !originalRequest._retry && !isRefreshRequest) {
      const refreshToken = localStorage.getItem('refreshToken')
      if (!refreshToken) {
        localStorage.clear()
        window.location = '/login'
        return Promise.reject(error)
      }

      if (isRefreshing) {
        // Queue this request until refresh completes
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        }).then(token => {
          originalRequest.headers.Authorization = `Bearer ${token}`
          return apiClient(originalRequest)
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        const { data } = await apiClient.post('/auth/refresh', { refresh_token: refreshToken })
        localStorage.setItem('token', data.token)
        if (data.refresh_token) localStorage.setItem('refreshToken', data.refresh_token)
        processQueue(null, data.token)
        originalRequest.headers.Authorization = `Bearer ${data.token}`
        return apiClient(originalRequest)
      } catch (refreshError) {
        processQueue(refreshError)
        localStorage.clear()
        window.location = '/login'
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  }
)

export default apiClient

// API methods
export const authAPI = {
  register: (data) => apiClient.post('/auth/register', data),
  login: (data) => apiClient.post('/auth/login', data),
  forgotPassword: (email) => apiClient.post('/auth/forgot-password', { email }),
  resetPassword: (token, newPassword) => apiClient.post('/auth/reset-password', { token, new_password: newPassword }),
  changePassword: (currentPassword, newPassword) =>
    apiClient.put('/settings/password', { current_password: currentPassword, new_password: newPassword }),
}

export const expensesAPI = {
  list: (params) => apiClient.get('/expenses', { params }),
  create: (data) => apiClient.post('/expenses', data),
  update: (id, data) => apiClient.put(`/expenses/${id}`, data),
  delete: (id) => apiClient.delete(`/expenses/${id}`),
  stats: (params) => apiClient.get('/expenses/stats', { params }),
}

export const categoriesAPI = {
  list: () => apiClient.get('/categories'),
  create: (data) => apiClient.post('/categories', data),
  update: (id, data) => apiClient.put(`/categories/${id}`, data),
  delete: (id) => apiClient.delete(`/categories/${id}`),
  createSubcategory: (catId, data) => apiClient.post(`/categories/${catId}/subcategories`, data),
  deleteSubcategory: (catId, subId) => apiClient.delete(`/categories/${catId}/subcategories/${subId}`),
}

export const balanceAPI = {
  get: (propertyId) => apiClient.get(`/properties/${propertyId}/balance`),
  details: (propertyId) => apiClient.get(`/properties/${propertyId}/balance/details`),
}

export const settlementsAPI = {
  create: (data) => apiClient.post('/settlements', data),
  list: (propertyId) => apiClient.get('/settlements', { params: { property_id: propertyId } }),
}

export const membersAPI = {
  list: (propertyId) => apiClient.get(`/properties/${propertyId}/members`),
  create: (propertyId, data) => apiClient.post(`/properties/${propertyId}/members`, data),
  get: (memberId) => apiClient.get(`/members/${memberId}`),
  update: (memberId, data) => apiClient.put(`/members/${memberId}`, data),
  delete: (memberId) => apiClient.delete(`/members/${memberId}`),
}

export const utilitiesAPI = {
  list: (params) => apiClient.get('/utilities', { params }),
  create: (data) => apiClient.post('/utilities', data),
  get: (id) => apiClient.get(`/utilities/${id}`),
  update: (id, data) => apiClient.put(`/utilities/${id}`, data),
  delete: (id) => apiClient.delete(`/utilities/${id}`),
  // Readings
  getReadings: (utilityId) => apiClient.get(`/utilities/${utilityId}/readings`),
  addReading: (utilityId, data) => apiClient.post(`/utilities/${utilityId}/readings`, data),
  updateReading: (utilityId, readingId, data) => apiClient.put(`/utilities/${utilityId}/readings/${readingId}`, data),
  deleteReading: (utilityId, readingId) => apiClient.delete(`/utilities/${utilityId}/readings/${readingId}`),
  // Bills
  getBills: (utilityId) => apiClient.get(`/utilities/${utilityId}/bills`),
  addBill: (utilityId, data) => apiClient.post(`/utilities/${utilityId}/bills`, data),
  updateBill: (utilityId, billId, data) => apiClient.put(`/utilities/${utilityId}/bills/${billId}`, data),
  updateBillFull: (utilityId, billId, data) => apiClient.put(`/utilities/${utilityId}/bills/${billId}/full`, data),
  deleteBill: (utilityId, billId) => apiClient.delete(`/utilities/${utilityId}/bills/${billId}`),
  // PDF uploads
  uploadBillPDF: (utilityId, file, templateId = null) => {
    const formData = new FormData()
    formData.append('pdf_file', file)
    if (templateId) formData.append('template_id', templateId)
    return apiClient.post(`/utilities/${utilityId}/bills/upload`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  uploadContractPDF: (file) => {
    const formData = new FormData()
    formData.append('pdf_file', file)
    return apiClient.post('/utilities/contract/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  // Reading comparison (autolettura vs lettura fornitore)
  compareReadings: (utilityId, baseThreshold = 2, thresholdPerDay = 1) =>
    apiClient.get(`/utilities/${utilityId}/compare-readings`, {
      params: { threshold: baseThreshold, threshold_per_day: thresholdPerDay }
    }),
  // Communications
  getCommunications: (utilityId) => apiClient.get(`/utilities/${utilityId}/communications`),
  addCommunication: (utilityId, data) => apiClient.post(`/utilities/${utilityId}/communications`, data),
  markCommunicationRead: (utilityId, commId) => apiClient.put(`/utilities/${utilityId}/communications/${commId}/read`),
  deleteCommunication: (utilityId, commId) => apiClient.delete(`/utilities/${utilityId}/communications/${commId}`),
}

export const communicationsAPI = {
  getAll: (params) => apiClient.get('/communications', { params }),
  getUnreadCount: () => apiClient.get('/communications/unread-count'),
}

export const templatesAPI = {
  // Bill templates
  listBillTemplates: () => apiClient.get('/templates/bills'),
  createBillTemplate: (data) => apiClient.post('/templates/bills', data),
  updateBillTemplate: (id, data) => apiClient.put(`/templates/bills/${id}`, data),
  deleteBillTemplate: (id) => apiClient.delete(`/templates/bills/${id}`),
}

export const projectsAPI = {
  list: (params) => apiClient.get('/projects', { params }),
  create: (data) => apiClient.post('/projects', data),
  get: (id) => apiClient.get(`/projects/${id}`),
  update: (id, data) => apiClient.put(`/projects/${id}`, data),
  delete: (id) => apiClient.delete(`/projects/${id}`),
}

export const exportAPI = {
  exportAll: () => apiClient.get('/export/all', { responseType: 'blob' }),
  exportExpenses: () => apiClient.get('/export/expenses', { responseType: 'blob' }),
  exportUtilities: () => apiClient.get('/export/utilities', { responseType: 'blob' }),
  exportProjects: () => apiClient.get('/export/projects', { responseType: 'blob' }),
  importData: (data) => apiClient.post('/import', data),
}

export const avatarAPI = {
  upload: (file) => {
    const formData = new FormData()
    formData.append('avatar', file)
    return apiClient.post('/settings/avatar', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  delete: () => apiClient.delete('/settings/avatar'),
}

export const adminAPI = {
  deleteUser: (userId) => apiClient.delete(`/admin/users/${userId}`),
  setUserRole: (userId, role) => apiClient.put(`/admin/users/${userId}/role`, { role }),
}

export const pdfAPI = {
  extractText: (file) => {
    const formData = new FormData()
    formData.append('pdf_file', file)
    return apiClient.post('/pdf/extract-text', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  // Extract text with word positions (for template wizard tooltips)
  extractTextWithPositions: (file) => {
    const formData = new FormData()
    formData.append('pdf_file', file)
    formData.append('with_positions', 'true')
    return apiClient.post('/pdf/extract-text', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  // Analyze PDF for template creation (Textract-like view)
  analyzePDF: (file) => {
    const formData = new FormData()
    formData.append('pdf_file', file)
    return apiClient.post('/pdf/analyze', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  // Cleanup temporary template images
  cleanupImages: (timestamp) => apiClient.delete(`/pdf/cleanup/${timestamp}`),
}
