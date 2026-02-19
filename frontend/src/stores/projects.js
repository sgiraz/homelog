import { defineStore } from 'pinia'
import { ref } from 'vue'
import { projectsAPI } from '@/api/client'

export const useProjectsStore = defineStore('projects', () => {
  const projects = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchProjects(propertyId, status = '') {
    loading.value = true
    error.value = null
    try {
      const params = { property_id: propertyId }
      if (status) params.status = status

      const { data } = await projectsAPI.list(params)
      projects.value = data || []
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createProject(project) {
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
    }
  }

  async function updateProject(id, project) {
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
    }
  }

  async function deleteProject(id) {
    try {
      await projectsAPI.delete(id)
      projects.value = projects.value.filter(p => p.id !== id)
    } catch (err) {
      error.value = err.message
      throw err
    }
  }

  return {
    projects,
    loading,
    error,
    fetchProjects,
    createProject,
    updateProject,
    deleteProject
  }
})
