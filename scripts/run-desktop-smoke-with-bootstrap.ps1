$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

$ErrorActionPreference = "Stop"

$env:GEN_CODE_UI_BASE_URL = if ($env:GEN_CODE_UI_BASE_URL) { $env:GEN_CODE_UI_BASE_URL } else { "http://127.0.0.1:5174/" }
$env:GEN_CODE_API_BASE_URL = if ($env:GEN_CODE_API_BASE_URL) { $env:GEN_CODE_API_BASE_URL } else { "http://127.0.0.1:10008" }
$env:GEN_CODE_ACCEPTANCE_MODE = "smoke"
$env:GOTOOLCHAIN = if ($env:GOTOOLCHAIN) { $env:GOTOOLCHAIN } else { "auto" }

$pythonCommand = Get-Command python -ErrorAction SilentlyContinue
if (-not $pythonCommand) {
  Write-Error "Setup failure: python is not available on PATH"
  exit 2
}

$npmCommand = Get-Command npm.cmd -ErrorAction SilentlyContinue
if (-not $npmCommand) {
  $npmCommand = Get-Command npm -ErrorAction SilentlyContinue
}
if (-not $npmCommand) {
  Write-Error "Setup failure: npm is not available on PATH"
  exit 2
}

$goCommand = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCommand) {
  Write-Error "Setup failure: go is not available on PATH"
  exit 2
}

$frontendRoot = Join-Path $projectRoot "desktop\frontend"
$artifactRoot = Join-Path $projectRoot "tmp\desktop-smoke-artifacts"
$frontendStdout = Join-Path $projectRoot "desktop\frontend\vite-smoke.stdout.log"
$frontendStderr = Join-Path $projectRoot "desktop\frontend\vite-smoke.stderr.log"
$serverStdout = Join-Path $projectRoot "server-smoke.stdout.log"
$serverStderr = Join-Path $projectRoot "server-smoke.stderr.log"

if (-not (Test-Path $artifactRoot)) {
  New-Item -ItemType Directory -Path $artifactRoot | Out-Null
}
$env:GEN_CODE_ARTIFACT_DIR = $artifactRoot

function Wait-HttpOk {
  param(
    [Parameter(Mandatory = $true)][string]$Url,
    [int]$TimeoutSeconds = 60
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $response = Invoke-WebRequest -UseBasicParsing $Url -TimeoutSec 5
      if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) {
        return
      }
    } catch {
    }
    Start-Sleep -Milliseconds 500
  }
  throw "Timed out waiting for $Url"
}

Write-Host "Desktop smoke bootstrap check"
Write-Host "  project root : $projectRoot"
Write-Host "  UI base URL  : $env:GEN_CODE_UI_BASE_URL"
Write-Host "  API base URL : $env:GEN_CODE_API_BASE_URL"

$serverProcess = $null
$frontendProcess = $null

try {
  $serverProcess = Start-Process -FilePath $goCommand.Source `
    -ArgumentList @("run", ".\cmd\server") `
    -WorkingDirectory $projectRoot `
    -RedirectStandardOutput $serverStdout `
    -RedirectStandardError $serverStderr `
    -WindowStyle Hidden `
    -PassThru

  $frontendProcess = Start-Process -FilePath $npmCommand.Source `
    -ArgumentList @("run", "dev", "--", "--host", "127.0.0.1", "--port", "5174") `
    -WorkingDirectory $frontendRoot `
    -RedirectStandardOutput $frontendStdout `
    -RedirectStandardError $frontendStderr `
    -WindowStyle Hidden `
    -PassThru

  Wait-HttpOk -Url "http://127.0.0.1:10008/api/runtime/status" -TimeoutSeconds 90
  Wait-HttpOk -Url "http://127.0.0.1:5174/" -TimeoutSeconds 90

  & $pythonCommand.Source (Join-Path $projectRoot "scripts\verify-desktop-live-refresh.py")
  $exitCode = $LASTEXITCODE
  if ($exitCode -ne 0) {
    Write-Error "Smoke verification failed with exit code $exitCode"
    exit $exitCode
  }
} finally {
  foreach ($proc in @($frontendProcess, $serverProcess)) {
    if ($null -ne $proc) {
      try {
        if (-not $proc.HasExited) {
          Stop-Process -Id $proc.Id -Force
        }
      } catch {
      }
    }
  }
}
