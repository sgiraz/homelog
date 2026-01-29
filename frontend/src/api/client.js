import axios from 'axios'

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1',
  headers: { 'Content-Type': 'application/json' }
})

// JWT interceptor
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// Refresh token interceptor
apiClient.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401) {
      const refreshToken = localStorage.getItem('refreshToken')
      if (refreshToken) {
        try {
          const { data } = await axios.post('/api/v1/auth/refresh', { refresh_token: refreshToken })
          localStorage.setItem('token', data.token)
          error.config.headers.Authorization = `Bearer ${data.token}`
          return axios(error.config)
        } catch {
          localStorage.clear()
          window.location = '/login'
        }
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
}

export const expensesAPI = {
  list: (params) => apiClient.get('/expenses', { params }),
  create: (data) => apiClient.post('/expenses', data),
  update: (id, data) => apiClient.put(`/expenses/${id}`, data),
  delete: (id) => apiClient.delete(`/expenses/${id}`),
  stats: (params) => apiClient.get('/expenses/stats', { params }),
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
