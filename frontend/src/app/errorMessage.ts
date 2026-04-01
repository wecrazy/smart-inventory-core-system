export function extractApiErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  if (!error || typeof error !== 'object') {
    return fallback;
  }

  if ('data' in error && error.data && typeof error.data === 'object') {
    const data = error.data as { error?: unknown; message?: unknown };

    if (typeof data.error === 'string' && data.error.trim() !== '') {
      return data.error;
    }

    if (typeof data.message === 'string' && data.message.trim() !== '') {
      return data.message;
    }
  }

  if ('error' in error && typeof error.error === 'string' && error.error.trim() !== '') {
    return error.error;
  }

  return fallback;
}