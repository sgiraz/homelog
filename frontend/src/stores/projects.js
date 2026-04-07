import { defineStore } from 'pinia'
import { ref } from 'vue'
import { projectsAPI } from '@/api/client'

export const useProjectsStore = defineStore('projects', () => {
  const projects = ref([])
  const loading = ref(false)
  const saving = ref(false)
  const error = ref(null)

  // Deduplication: track in-flight fetch
  let fetchPromise = null

  async function fetchProjects(propertyId, status = '') {
    // Deduplicate: if already fetching, return existing promise
    if (fetchPromise) return fetchPromise

    loading.value = true
    error.value = null

    fetchPromise = projectsAPI.list({ property_id: propertyId, ...(status && { status }) })
      .then(({ data }) => {
        projects.value = data || []
      })
      .catch(err => {
        error.value = err.message
        throw err
      })
      .finally(() => {
        loading.value = false
        fetchPromise = null
      })

    return fetchPromise
  }

  async function fetchProject(id) {
    loading.value = true
    error.value = null
    try {
      const { data } = await projectsAPI.get(id)
      // Update in local list if present
      const index = projects.value.findIndex(p => p.id === id)
      if (index !== -1) {
        projects.value[index] = data
      }
      return data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createProject(project) {
    saving.value = true
    error.value = null
    try {
      const { data } = await projectsAPI.create(project)
      // Backend Create returns raw project without stats - add defaults
      data.stats = data.stats || {
        total_budget: data.budget || 0,
        total_spent: 0,
        remaining: data.budget || 0,
        percentage_spent: 0,
        expense_count: 0
      }
      projects.value.push(data)
      return data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      saving.value = false
    }
  }

  async function updateProject(id, project) {
    saving.value = true
    error.value = null
    try {
      const { data } = await projectsAPI.update(id, project)
      const index = projects.value.findIndex(p => p.id === id)
      if (index !== -1) {
        projects.value[index] = data
      }
      return data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      saving.value = false
    }
  }

  async function deleteProject(id) {
    saving.value = true
    error.value = null
    try {
      await projectsAPI.delete(id)
      projects.value = projects.value.filter(p => p.id !== id)
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      saving.value = false
    }
  }

  return {
    projects,
    loading,
    saving,
    error,
    fetchProjects,
    fetchProject,
    createProject,
    updateProject,
    deleteProject
  }
})
