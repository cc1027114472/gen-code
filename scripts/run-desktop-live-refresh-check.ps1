$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

$ErrorActionPreference = "Stop"

$env:GEN_CODE_UI_BASE_URL = if ($env:GEN_CODE_UI_BASE_URL) { $env:GEN_CODE_UI_BASE_URL } else { "http://127.0.0.1:5174/" }
$env:GEN_CODE_API_BASE_URL = if ($env:GEN_CODE_API_BASE_URL) { $env:GEN_CODE_API_BASE_URL } else { "http://127.0.0.1:10008" }
$env:GEN_CODE_ACCEPTANCE_MODE = "full"
$env:GOTOOLCHAIN = if ($env:GOTOOLCHAIN) { $env:GOTOOLCHAIN } else { "auto" }
$scriptPath = Join-Path $projectRoot "scripts\verify-desktop-live-refresh.py"
$baselineLane = "canonical remote browser acceptance with desktop copy/runtime checks (5174 + 10008)"

Write-Host "Desktop live refresh and copy/runtime alignment check"
Write-Host "  project root : $projectRoot"
Write-Host "  lane         : $baselineLane"
Write-Host "  script       : $scriptPath"
Write-Host "  UI base URL  : $env:GEN_CODE_UI_BASE_URL"
Write-Host "  API base URL : $env:GEN_CODE_API_BASE_URL"
Write-Host "  mode         : $env:GEN_CODE_ACCEPTANCE_MODE"
Write-Host "  failures     : remote canonical live matrix + fallback evidence-only"

if (-not (Test-Path $scriptPath)) {
  Write-Error "Setup failure: verify-desktop-live-refresh.py not found at $scriptPath"
  exit 2
}

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
$frontendStdout = Join-Path $projectRoot "desktop\frontend\vite-full.stdout.log"
$frontendStderr = Join-Path $projectRoot "desktop\frontend\vite-full.stderr.log"
$serverStdout = Join-Path $projectRoot "server-full.stdout.log"
$serverStderr = Join-Path $projectRoot "server-full.stderr.log"

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
        return $true
      }
    } catch {
    }
    Start-Sleep -Milliseconds 500
  }
  return $false
}

function Test-MCPBaselineReady {
  param(
    [Parameter(Mandatory = $true)][string]$ApiBaseUrl
  )

  try {
    $payload = Invoke-WebRequest -UseBasicParsing "$ApiBaseUrl/api/mcp/servers" -TimeoutSec 5
    $json = $payload.Content | ConvertFrom-Json
    $items = @($json.data.items)
    $serverIds = @($items | ForEach-Object { $_.id })
    $required = @("external-fixture", "sdk-external-fixture", "third-party-time")
    $missing = @($required | Where-Object { $_ -notin $serverIds })
    return @{
      Ready = ($missing.Count -eq 0)
      Missing = $missing
      ServerIds = $serverIds
    }
  } catch {
    return @{
      Ready = $false
      Missing = @("external-fixture", "sdk-external-fixture", "third-party-time")
      ServerIds = @()
    }
  }
}

$serverProcess = $null
$frontendProcess = $null
$bootstrapStarted = $false

try {
  $apiReady = Wait-HttpOk -Url "$($env:GEN_CODE_API_BASE_URL)/api/runtime/status" -TimeoutSeconds 5
  $uiReady = Wait-HttpOk -Url $env:GEN_CODE_UI_BASE_URL -TimeoutSeconds 5
  $baseline = if ($apiReady) { Test-MCPBaselineReady -ApiBaseUrl $env:GEN_CODE_API_BASE_URL } else { @{ Ready = $false; Missing = @("external-fixture", "sdk-external-fixture", "third-party-time"); ServerIds = @() } }

  if (-not ($apiReady -and $uiReady -and $baseline.Ready)) {
    Write-Host "  bootstrap    : starting current-code server/frontend because existing canonical instance is missing or incomplete"
    if ($apiReady -and -not $baseline.Ready) {
      Write-Host ("  baseline     : existing API instance missing MCP lanes: " + ($baseline.Missing -join ", "))
    }

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

    if (-not (Wait-HttpOk -Url "$($env:GEN_CODE_API_BASE_URL)/api/runtime/status" -TimeoutSeconds 90)) {
      throw "Timed out waiting for $($env:GEN_CODE_API_BASE_URL)/api/runtime/status"
    }
    if (-not (Wait-HttpOk -Url $env:GEN_CODE_UI_BASE_URL -TimeoutSeconds 90)) {
      throw "Timed out waiting for $($env:GEN_CODE_UI_BASE_URL)"
    }

    $baseline = Test-MCPBaselineReady -ApiBaseUrl $env:GEN_CODE_API_BASE_URL
    if (-not $baseline.Ready) {
      throw ("Bootstrapped canonical instance still missing MCP lanes: " + ($baseline.Missing -join ", "))
    }
    $bootstrapStarted = $true
  } else {
    Write-Host "  bootstrap    : using existing canonical instance with complete MCP baseline"
  }

  & $pythonCommand.Source $scriptPath
  $exitCode = $LASTEXITCODE
  if ($exitCode -ne 0) {
    Write-Error "Verification failed with exit code $exitCode"
    exit $exitCode
  }
} finally {
  if ($bootstrapStarted) {
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
}
