#!/bin/sh
# Cache Vent installer for macOS and Linux.
#
#   curl -fsSL __BASE__/install.sh | sh
#   curl -fsSL __BASE__/install.sh | sh -s -- --uninstall
#
# macOS: builds "Cache Vent.app" in /Applications (or ~/Applications), signs it ad hoc
# on this Mac and clears the download flag, so it opens without an Apple Developer ID.
# Linux: installs ~/.local/bin/cachevent plus a launcher in your applications menu.
# Both: the command line tool is ~/.local/bin/cachevent. Checksums are verified.
# Nothing here needs sudo. Your scans and quarantine are never touched.
set -eu

BASE="${CACHEVENT_BASE:-__BASE__}"
BIN_DIR="${CACHEVENT_BIN_DIR:-$HOME/.local/bin}"
APP_NAME="Cache Vent"

say() { printf '%s\n' "$*"; }
die() { printf 'cachevent install: %s\n' "$*" >&2; exit 1; }
has() { command -v "$1" >/dev/null 2>&1; }

fetch() { # url dest
  if has curl; then curl -fsSL "$1" -o "$2"
  elif has wget; then wget -q "$1" -O "$2"
  else die "needs curl or wget"; fi
}

sha256() {
  if has sha256sum; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

os=$(uname -s)
case "$os" in
  Darwin) os=macos ;;
  Linux) os=linux ;;
  *) die "unsupported system: $os (Windows: see $BASE)" ;;
esac

arch=$(uname -m)
case "$arch" in
  arm64 | aarch64) arch=arm64 ;;
  x86_64 | amd64) arch=x64 ;;
  *) die "unsupported processor: $arch" ;;
esac
# A shell running under Rosetta reports x86_64 on Apple silicon; install the native build.
if [ "$os" = macos ] && [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" = 1 ]; then arch=arm64; fi

app_dir() {
  for d in /Applications "$HOME/Applications"; do
    if [ -d "$d/$APP_NAME.app" ]; then printf '%s' "$d"; return; fi
  done
  if [ -w /Applications ]; then printf '/Applications'; else printf '%s' "$HOME/Applications"; fi
}

if [ "${1:-}" = "--uninstall" ]; then
  rm -f "$BIN_DIR/cachevent"
  if [ "$os" = macos ]; then
    rm -rf "/Applications/$APP_NAME.app" "$HOME/Applications/$APP_NAME.app"
  else
    rm -f "$HOME/.local/share/applications/cachevent.desktop" "$HOME/.local/share/icons/hicolor/256x256/apps/cachevent.png"
  fi
  say "Cache Vent removed. Your scan history and quarantine are kept in its data folder; run 'cachevent purge --all' first if you want those gone too."
  exit 0
fi

file="cachevent-$os-$arch"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

say "Downloading $file ..."
fetch "$BASE/downloads/$file" "$tmp/cachevent"
fetch "$BASE/downloads/SHA256SUMS.txt" "$tmp/sums"
want=$(grep " $file\$" "$tmp/sums" | cut -d' ' -f1)
[ -n "$want" ] || die "no checksum published for $file"
[ "$(sha256 "$tmp/cachevent")" = "$want" ] || die "checksum mismatch for $file; nothing was installed"
chmod 755 "$tmp/cachevent"
"$tmp/cachevent" version >/dev/null 2>&1 || die "the downloaded build does not run on this system ($os $arch)"

mkdir -p "$BIN_DIR"

if [ "$os" = macos ]; then
  dir=$(app_dir)
  app="$dir/$APP_NAME.app"
  mkdir -p "$dir"
  rm -rf "$app"
  mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
  mv "$tmp/cachevent" "$app/Contents/MacOS/cachevent"
  # The app's entry point opens the interface; the same binary is the CLI.
  cat >"$app/Contents/MacOS/launch" <<'EOF'
#!/bin/sh
exec "$(dirname "$0")/cachevent" ui
EOF
  chmod 755 "$app/Contents/MacOS/launch"
  fetch "$BASE/downloads/cachevent.icns" "$app/Contents/Resources/cachevent.icns" || true
  cat >"$app/Contents/Info.plist" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Cache Vent</string>
  <key>CFBundleDisplayName</key><string>Cache Vent</string>
  <key>CFBundleIdentifier</key><string>app.cachevent</string>
  <key>CFBundleExecutable</key><string>launch</string>
  <key>CFBundleIconFile</key><string>cachevent</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>__VERSION__</string>
  <key>CFBundleVersion</key><string>__VERSION__</string>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
  <key>LSUIElement</key><true/>
</dict>
</plist>
EOF
  # Not notarized: sign ad hoc on this Mac and drop the quarantine flag so Gatekeeper lets it open.
  codesign --force --deep --sign - "$app" >/dev/null 2>&1 || die "codesign failed (install the Xcode command line tools: xcode-select --install)"
  xattr -dr com.apple.quarantine "$app" 2>/dev/null || true
  ln -sf "$app/Contents/MacOS/cachevent" "$BIN_DIR/cachevent"
  say "Installed $app"
  say "Open it from Launchpad or Spotlight (Cache Vent), or run: open \"$app\""
else
  mv "$tmp/cachevent" "$BIN_DIR/cachevent"
  apps="$HOME/.local/share/applications"
  icons="$HOME/.local/share/icons/hicolor/256x256/apps"
  mkdir -p "$apps" "$icons"
  fetch "$BASE/downloads/cachevent.png" "$icons/cachevent.png" || true
  cat >"$apps/cachevent.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Cache Vent
Comment=See what your dev tools keep on disk and free what is safe
Exec="$BIN_DIR/cachevent" ui
Icon=cachevent
Terminal=false
Categories=Utility;System;
EOF
  has update-desktop-database && update-desktop-database "$apps" >/dev/null 2>&1 || true
  say "Installed $BIN_DIR/cachevent and a Cache Vent launcher in your applications menu."
  say "Start it from the menu, or run: cachevent ui"
fi

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) say "Add $BIN_DIR to your PATH to use the cachevent command: echo 'export PATH=\"$BIN_DIR:\$PATH\"' >> ~/.profile" ;;
esac
say "The interface opens in a Chrome, Edge or Brave window if one is installed, otherwise in your default browser."
