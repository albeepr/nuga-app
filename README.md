# Nuga — Air96 V2 edition

> **This is a fork of [mishamyrt/Nuga](https://github.com/mishamyrt/Nuga)** that adds support for the
> **NuPhy Air96 V2** (QMK/VIA firmware). All credit for Nuga goes to its original author;
> Halo keyboards work exactly as in upstream Nuga 2.3.2.

Application for controlling NuPhy® keyboards.

<img src="https://nuga.myrt.co/window.png" width="800px" />

### Features

- Lights control.
- Key mapping control (Halo75, Halo65 and Air96 V2).
- Macros control.

**Supported devices**: Halo75, Halo65, Halo96, Air96 V2.

### NuPhy Air96 V2 (QMK/VIA)

The Air96 V2 runs QMK firmware and is controlled over the VIA raw HID protocol
instead of the Halo protocol. Supported:

- Backlight: all 40 effects, brightness, speed, color.
- Key remapping on the Mac and Windows layers, including every key's Fn layer, plus NuPhy actions (Bluetooth/2.4G switching, side-light keys, battery display).
- Macros (VIA format).
- Save, load and restore default.

Not supported: side-light settings (NuPhy's firmware only changes them with Fn shortcuts and does not expose them over USB).

Tested on an Air96 V2 ANSI with NuPhy's shipping firmware (version 4, USB ID `19F5:3265`).
USB ID `19F5:3266` (NuPhy's public QMK source) is also recognised.

Notes:

- Keys Nuga has no name for (layer toggles, etc.) show as "QMK code 0x…" and are never overwritten unless you change them.
- "Restore default" restores the keyboard firmware's own default keymap and clears macros.
- The per-effect color palette is stored on the computer in `~/Library/Application Support/nuga/Air96V2-palette.json`; the keyboard only stores the active color.
- Macro steps must be ordinary keys (letters, modifiers, media...). NuPhy-specific actions can't go inside macros (VIA limitation).
- Close usevia.app in the browser while using Nuga.

#### Wired, Bluetooth and 2.4G

Settings travel over the USB cable only: the keyboard's radio chip runs NuPhy's
closed firmware, which carries keystrokes but not configuration traffic. Switching
to Bluetooth or 2.4G does not turn USB off, though, so you can type wirelessly
with a USB cable plugged into the Mac and Nuga talks to the keyboard over the cable.
This follows from the firmware source but has not been tested on hardware yet.

The modified Nuga library lives in `third_party/nuga-lib` (wired in via `replace` in `go.mod`).

#### Building this fork

Needs Go (current Homebrew version is fine), the latest Wails v2 CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@latest`), Node and `brew install hidapi`.
The UI needs pnpm 9 (pnpm 11 blocks esbuild's install script):

```sh
wails generate module
(cd ui && npx -y pnpm@9.15.9 install --frozen-lockfile && npx -y pnpm@9.15.9 run build)
wails build -s -clean -trimpath -platform darwin/arm64
```

## Installation

### macOS

1. Download `Nuga-2.3.2-air96v2-mac-arm64.zip` from this fork's [releases](https://github.com/albeepr/nuga-app/releases) (Apple Silicon Macs).
2. Unzip the archive, copy `Nuga.app` to Applications.
3. Launch `Nuga.app`. The build is not signed: the first time, right-click it and choose **Open**.
4. The application requests permission to Input Monitoring. This is required to work with the USB HID device. Grant permission.
5. Restart the app

## Development

For local development, you need to set up environments. For this purpose you will need:

- [golang](https://go.dev/doc/install) 1.20+
- [Wails](https://wails.io/docs/gettingstarted/installation) 2.5+
- [pnpm](https://pnpm.io/installation) 8+

### Libraries

#### macOS

```sh
brew install hidapi
```

### Starting

To run a local application, the target `dev` is used:

```sh
make dev
```

If you want to run another OS interface on macOS, you should run the `dev-universal` target:

```sh
make dev-universal
```

### Build

Cross-compiling the application is not possible at this time. Therefore, full build is only possible on macOS. A native build for macOS can be built on macs with Intel® and Apple Silicon™ processors , Linux version is built on either macOS (via Docker) or the amd64 version of Linux.

For native builds, there are targets in the Makefile:

- `build/darwin` – To build macOS app (arm64, amd64)
- `build/linux` – To build Linux app binary (arm64, amd64)

The `build` command is a common alias for these commands and performs a build on the OS that is currently in use.

```sh
make build
```

The result of the build will be archives with the application in the `dist` or `app/build/bin` folder.

### Release

To publish a new release, set version in [Makefile](./Makefile) and run:

```
make publish
```

Automatic build, signing and notarization will be started. Once the build is complete, the files will be attached to the created GitHub Release.

The release description is compiled automatically from the [CHANGELOG.md](./CHANGELOG.md) file.

### Linux cross-compilation

The Linux version can be built on macOS using Docker. To build an image with the toolkit, run the command:

```sh
make linux-builder/setup
```

To build a Linux AppImage using the resulting image, run the command:

```sh
make linux-builder/release
```

### Hooks

It is highly recommended to install git hooks in the project repository. They make it easier to follow code Style. To install them, run the command:

```sh
make setup-qa
make hook
```

## Trademarks

NuPhy® is a registered trademark of NuPhy Studio. Nuga is an unofficial product and is not affiliated with NuPhy Studio.
