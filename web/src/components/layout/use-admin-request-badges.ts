import { useEffect, useState } from 'react'
import { listLicenseRequests } from '../../lib/license-request-api'
import { listMaintenanceRequests } from '../../lib/maintenance-api'
import {
  countAdminRequestBadges,
  type AdminRequestBadges,
} from './admin-request-badges'

const REFRESH_EVENT = 'license-manager:admin-request-badges-refresh'
const REFRESH_INTERVAL_MS = 30_000
const EMPTY_BADGES: AdminRequestBadges = {
  licenseRequests: 0,
  maintenanceRequests: 0,
}

export function notifyAdminRequestBadgesChanged() {
  window.dispatchEvent(new Event(REFRESH_EVENT))
}

export function useAdminRequestBadges(accessToken: string): AdminRequestBadges {
  const [badges, setBadges] = useState<AdminRequestBadges>(EMPTY_BADGES)

  useEffect(() => {
    let active = true
    let requestSequence = 0

    async function load() {
      const sequence = ++requestSequence
      try {
        const [licenseRequests, maintenanceRequests] = await Promise.all([
          listLicenseRequests(accessToken, { status: 'pending' }),
          listMaintenanceRequests(accessToken, { status: 'pending' }),
        ])
        if (!active || sequence !== requestSequence) return
        setBadges(countAdminRequestBadges(
          licenseRequests.items.map((item) => item.status),
          maintenanceRequests.items.map((item) => item.status),
        ))
      } catch {
        // Keep the last successful counts while the backend is temporarily unavailable.
      }
    }

    const refresh = () => { void load() }
    const refreshWhenVisible = () => {
      if (document.visibilityState === 'visible') refresh()
    }

    refresh()
    const timer = window.setInterval(refreshWhenVisible, REFRESH_INTERVAL_MS)
    window.addEventListener('focus', refresh)
    window.addEventListener(REFRESH_EVENT, refresh)
    document.addEventListener('visibilitychange', refreshWhenVisible)

    return () => {
      active = false
      requestSequence++
      window.clearInterval(timer)
      window.removeEventListener('focus', refresh)
      window.removeEventListener(REFRESH_EVENT, refresh)
      document.removeEventListener('visibilitychange', refreshWhenVisible)
    }
  }, [accessToken])

  return badges
}
