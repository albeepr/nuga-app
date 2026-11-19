import { GetStatus } from '$wails/go/usecase/DeviceUsecase'

import type { KeyboardStatus } from '../model/types'

export async function getStatus (): Promise<KeyboardStatus | null> {
  const status: KeyboardStatus | null = await GetStatus()
  return status ?? null
}
