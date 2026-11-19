export type Supports = {
  backlight: boolean
  sidelight: boolean
  halolight: boolean
  keys: boolean
  state: boolean
}

export type KeyboardStatus = {
  battery: number
  charging: boolean
  connection: string
  os: string
}
