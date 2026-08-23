export interface NormalizedAPIError {
  message: string
  status: number
  code?: string
}

export function normalizeAPIError(error: unknown, fallback: string): NormalizedAPIError {
  if (!(error instanceof Error)) return { message: fallback, status: 0 }

  const candidate = error as Error & { status?: unknown; code?: unknown }
  const normalized: NormalizedAPIError = {
    message: candidate.message || fallback,
    status: typeof candidate.status === 'number' ? candidate.status : 0,
  }
  if (typeof candidate.code === 'string' && candidate.code) normalized.code = candidate.code
  return normalized
}

export function isNoAvailableSeatsConflict(error: NormalizedAPIError): boolean {
  return error.status === 409 && error.code === 'no_available_seats'
}
