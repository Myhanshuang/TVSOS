import request from '@/utils/request'

// ---------- 沙箱 ----------
export const listSandboxes = (archived = false) =>
    request.get('/sandboxes', { params: archived ? { archived: 1 } : {} })
export const createSandbox = (data) => request.post('/sandboxes', data)
export const getSandbox = (id) => request.get(`/sandboxes/${id}`)
export const updateSandbox = (id, data) => request.put(`/sandboxes/${id}`, data)
export const archiveSandbox = (id) => request.delete(`/sandboxes/${id}`)

// ---------- 实验方案 ----------
export const listSchemes = (sandboxId) => request.get(`/sandboxes/${sandboxId}/schemes`)
export const createScheme = (sandboxId, data) => request.post(`/sandboxes/${sandboxId}/schemes`, data)
export const updateScheme = (schemeId, data) => request.put(`/schemes/${schemeId}`, data)
export const deleteScheme = (schemeId) => request.delete(`/schemes/${schemeId}`)

// ---------- 实验（批次） ----------
export const listExperiments = (sandboxId) => request.get(`/sandboxes/${sandboxId}/experiments`)
export const createExperiment = (sandboxId, data) => request.post(`/sandboxes/${sandboxId}/experiments`, data)
export const getExperiment = (id) => request.get(`/experiments/${id}`)
export const updateExperimentParams = (id, data) => request.put(`/experiments/${id}/params`, data)
export const generateExperimentData = (id) => request.post(`/experiments/${id}/generate`)
export const startExperiment = (id) => request.post(`/experiments/${id}/start`)
export const stopExperiment = (id) => request.post(`/experiments/${id}/stop`)
export const getExperimentStatus = (id) => request.get(`/experiments/${id}/status`)
export const exportExperiment = (id) => request.get(`/experiments/${id}/export`)
export const importExperiment = (sandboxId, data) => request.post(`/sandboxes/${sandboxId}/import`, data)
export const getExperimentResult = (id) => request.get(`/experiments/${id}/result`)
export const compareExperiments = (expIds) =>
    request.get('/compare', { params: { exp_ids: expIds.join(',') } })
