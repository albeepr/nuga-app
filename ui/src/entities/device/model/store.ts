import { createEffect, createEvent, createStore, sample } from 'effector'
import { interval, or } from 'patronum'

import { connected, createHIDEffect, disconnected } from '$shared/model'

import { getFirmware, getStatus, getSupports } from '../api'
import { restoreDefaultState, restoreState, saveState } from '../api/state'
import { defaultSupports } from '../lib/constants'
import type { KeyboardStatus } from './types'

export const firmwareVersionUpdated = createEvent<string>('firmwareVersionUpdated')
export const userStateSaved = createEvent('stateSaved')
export const userStateRestored = createEvent('stateRestored')
export const defaultStateRestored = createEvent('defaultStateRestored')
// Combined events. Triggers AFTER state effects
export const anyStateRestored = createEvent('anyStateRestored')

export const supportsStore = createStore(defaultSupports, { name: 'supports' })
export const firmwareVersionStore = createStore('dev', {
  name: 'firmwareVersion',
})

export const getFirmwareFx = createEffect('getFirmware', {
  handler: getFirmware,
})
export const getSupportsFx = createEffect('getSupports', {
  handler: getSupports,
})
export const saveStateFx = createHIDEffect({
  name: 'saveStateFx',
  handler: saveState,
})
export const restoreUserStateFx = createHIDEffect({
  name: 'restoreUserStateFx',
  handler: restoreState,
})
export const restoreDefaultStateFx = createHIDEffect({
  name: 'restoreDefaultStateFx',
  handler: restoreDefaultState,
})

sample({
  clock: getFirmwareFx.doneData,
  target: firmwareVersionUpdated,
})

firmwareVersionStore.on(firmwareVersionUpdated, (_, value) => value)
supportsStore.on(getSupportsFx.doneData, (_, value) => value)

sample({
  clock: connected,
  target: [getFirmwareFx, getSupportsFx],
})

sample({
  clock: userStateSaved,
  target: saveStateFx,
})
sample({
  clock: userStateRestored,
  target: restoreUserStateFx,
})
sample({
  clock: defaultStateRestored,
  target: restoreDefaultStateFx,
})
sample({
  clock: [restoreUserStateFx.done, restoreDefaultStateFx.done],
  target: anyStateRestored,
})

export const restoringStateStore = or(
  restoreUserStateFx.pending,
  restoreDefaultStateFx.pending,
)

// Live status (battery, connection) for keyboards that report it
export const statusStore = createStore<KeyboardStatus | null>(null, {
  name: 'keyboardStatus',
})
export const getStatusFx = createEffect('getStatus', { handler: getStatus })

const { tick: statusTick } = interval({
  timeout: 30000,
  start: connected,
  stop: disconnected,
  leading: true,
})

sample({
  clock: statusTick,
  target: getStatusFx,
})
statusStore.on(getStatusFx.doneData, (_, value) => value)
statusStore.reset(disconnected)
