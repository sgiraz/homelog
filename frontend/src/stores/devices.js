import { defineStore } from 'pinia'
import { ref } from 'vue'
import { devicesAPI, deviceCategoriesAPI } from '@/api/client'


export const useDevicesStore = defineStore('devices', () => {
    const devices = ref([])
    const categories = ref([])
    const loading = ref(false)
    const saving = ref(false)
    const error = ref(null)
    const total = ref(0)
    const limit = ref(50)
    const offset = ref(0)

    async function fetchDevices(params = {}) {
        loading.value = true
        error.value = null

        try {
            const { data } = await devicesAPI.list(params)
            devices.value = data.devices || []
            total.value = data.total || 0
            limit.value = data.limit || 50
            offset.value = data.offset || 0
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            loading.value = false
        }
    }
    async function fetchDevice(id) {
        loading.value = true
        error.value = null

        try {
            const { data } = await devicesAPI.get(id)
            return data
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            loading.value = false
        }

    }
    async function fetchCategories() {
        try {
            const { data } = await deviceCategoriesAPI.list()
            categories.value = data || []
            return categories.value
        } catch (err) {
            error.value = err.message
            throw err
        }
    }
    async function createCategory(data) {
        saving.value = true
        error.value = null

        try {
            const { data: createdCategory } = await deviceCategoriesAPI.create(data)
            return createdCategory
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            saving.value = false
        }
    }
    async function updateCategory(id, data) {
        saving.value = true
        error.value = null

        try {
            const { data: updatedCategory } = await deviceCategoriesAPI.update(id, data)
            return updatedCategory
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            saving.value = false
        }
    }
    async function deleteCategory(id) {
        saving.value = true
        error.value = null

        try {
            await deviceCategoriesAPI.delete(id)
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            saving.value = false
        }
    }
    async function createDevice(data) {
        saving.value = true
        error.value = null

        try {
            const { data: createdDevice } = await devicesAPI.create(data)
            return createdDevice
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            saving.value = false
        }
    }
    async function updateDevice(id, data) {
        saving.value = true
        error.value = null

        try {
            const { data: updatedDevice } = await devicesAPI.update(id, data)
            return updatedDevice
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            saving.value = false
        }
    }
    async function deleteDevice(id) {
        saving.value = true
        error.value = null

        try {
            await devicesAPI.delete(id)
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            saving.value = false
        }
    }

    return {
        devices,
        categories,
        loading,
        saving,
        error,
        total,
        limit,
        offset,
        fetchDevices,
        fetchDevice,
        fetchCategories,
        updateDevice,
        createDevice,
        deleteDevice,
        createCategory,
        updateCategory,
        deleteCategory,
    }


})
