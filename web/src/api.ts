const TOKEN_KEY = 'ops_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}
export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t)
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(path, { ...init, headers: { ...headers, ...(init?.headers || {}) } })
  if (res.status === 401) {
    clearToken()
    window.location.href = '/login'
    throw new Error('未登录')
  }
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new Error(body?.error || `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

export interface NetworkInterface {
  name: string
  mac: string
  ips: string[]
}

export interface Device {
  id: number
  device_id: string
  hostname: string
  os_name: string
  os_version: string
  os_build: string
  agent_version: string
  logged_in_user: string
  status: string
  last_seen_at: string | null
  first_seen_at: string
  interfaces: NetworkInterface[]
}
