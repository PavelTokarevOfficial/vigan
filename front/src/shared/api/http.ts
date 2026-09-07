export async function readError(response: Response, fallback: string) {
  try {
    const payload = await response.json()
    return payload.error?.message || fallback
  } catch {
    return fallback
  }
}

export async function readData<T>(response: Response): Promise<T> {
  if (!response.ok) throw new Error(await readError(response, 'Ошибка запроса'))
  return (await response.json()).data as T
}
