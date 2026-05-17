$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

$env:GEN_CODE_UI_BASE_URL = if ($env:GEN_CODE_UI_BASE_URL) { $env:GEN_CODE_UI_BASE_URL } else { "http://127.0.0.1:5174/" }
$env:GEN_CODE_API_BASE_URL = if ($env:GEN_CODE_API_BASE_URL) { $env:GEN_CODE_API_BASE_URL } else { "http://127.0.0.1:10008" }
$env:GEN_CODE_ACCEPTANCE_MODE = "smoke"
$scriptPath = Join-Path $projectRoot "scripts\verify-desktop-live-refresh.py"
$baselineLane = "canonical remote browser smoke acceptance (5174 + 10008)"

Write-Host "Desktop smoke copy/runtime alignment check"
Write-Host "  project root : $projectRoot"
Write-Host "  lane         : $baselineLane"
Write-Host "  script       : $scriptPath"
Write-Host "  UI base URL  : $env:GEN_CODE_UI_BASE_URL"
Write-Host "  API base URL : $env:GEN_CODE_API_BASE_URL"
Write-Host "  mode         : $env:GEN_CODE_ACCEPTANCE_MODE"
Write-Host "  fallback     : not part of the smoke gate"

if (-not (Test-Path $scriptPath)) {
  Write-Error "Setup failure: verify-desktop-live-refresh.py not found at $scriptPath"
  exit 2
}

$pythonCommand = Get-Command python -ErrorAction SilentlyContinue
if (-not $pythonCommand) {
  Write-Error "Setup failure: python is not available on PATH"
  exit 2
}

& $pythonCommand.Source $scriptPath
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
  Write-Error "Smoke verification failed with exit code $exitCode"
  exit $exitCode
}
