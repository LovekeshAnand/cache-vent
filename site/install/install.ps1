# Cache Vent installer for Windows.
#
#   irm __BASE__/install.ps1 | iex
#
# Installs CacheVent.exe for the current user (no admin), adds a Start menu shortcut and
# opens the app. Downloading this way skips the SmartScreen prompt a browser download gets.
# The checksum is verified first. To remove: delete the folder and the shortcut named below.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$base = if ($env:CACHEVENT_BASE) { $env:CACHEVENT_BASE } else { '__BASE__' }
$dir = if ($env:CACHEVENT_DIR) { $env:CACHEVENT_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\Cache Vent' }
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64') { 'arm64' } else { 'x64' }
$file = "CacheVent-windows-$arch.exe"
$exe = Join-Path $dir 'CacheVent.exe'

Write-Host "Downloading $file ..."
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("cachevent-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Invoke-WebRequest -UseBasicParsing -Uri "$base/downloads/$file" -OutFile "$tmp\$file"
  $sums = (Invoke-WebRequest -UseBasicParsing -Uri "$base/downloads/SHA256SUMS.txt").Content
  if ($sums -is [byte[]]) { $sums = [Text.Encoding]::UTF8.GetString($sums) }
  $line = $sums -split "`n" | Where-Object { $_.Trim().EndsWith(" $file") } | Select-Object -First 1
  if (-not $line) { throw "No checksum is published for $file." }
  $want = ($line -split '\s+')[0].ToLower()
  $got = (Get-FileHash -Algorithm SHA256 "$tmp\$file").Hash.ToLower()
  if ($got -ne $want) { throw "Checksum mismatch for $file. Nothing was installed." }

  Get-Process CacheVent -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $exe } | Stop-Process -Force
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
  Move-Item -Force "$tmp\$file" $exe
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

if (-not $env:CACHEVENT_NO_SHORTCUT) {
  $lnk = Join-Path ([Environment]::GetFolderPath('Programs')) 'Cache Vent.lnk'
  $sh = (New-Object -ComObject WScript.Shell).CreateShortcut($lnk)
  $sh.TargetPath = $exe
  $sh.WorkingDirectory = $dir
  $sh.Description = 'See what your dev tools keep on disk and free what is safe'
  $sh.Save()
  Write-Host "Added a Start menu shortcut: $lnk"
}

Write-Host "Installed $exe"
if (-not $env:CACHEVENT_NO_LAUNCH) { Start-Process $exe }
