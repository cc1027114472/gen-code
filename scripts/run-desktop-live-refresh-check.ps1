$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$env:GEN_CODE_UI_BASE_URL = if ($env:GEN_CODE_UI_BASE_URL) { $env:GEN_CODE_UI_BASE_URL } else { "http://127.0.0.1:5174/" }
$env:GEN_CODE_API_BASE_URL = if ($env:GEN_CODE_API_BASE_URL) { $env:GEN_CODE_API_BASE_URL } else { "http://127.0.0.1:10008" }
$env:GEN_CODE_ACCEPTANCE_MODE = "full"
$env:GEN_CODE_BROWSER_ONLY_ACCEPTANCE = if ($env:GEN_CODE_BROWSER_ONLY_ACCEPTANCE) { $env:GEN_CODE_BROWSER_ONLY_ACCEPTANCE } else { "0" }
$env:GEN_CODE_BROWSER_PUBLIC_WEB_MODE = if ($env:GEN_CODE_BROWSER_PUBLIC_WEB_MODE) { $env:GEN_CODE_BROWSER_PUBLIC_WEB_MODE } else { "required" }
$env:GEN_CODE_BROWSER_PUBLIC_TARGETS = if ($env:GEN_CODE_BROWSER_PUBLIC_TARGETS) { $env:GEN_CODE_BROWSER_PUBLIC_TARGETS } else { "https://example.com/,https://www.iana.org/domains/reserved" }
$env:GEN_CODE_BROWSER_PUBLIC_BASE_URL = if ($env:GEN_CODE_BROWSER_PUBLIC_BASE_URL) { $env:GEN_CODE_BROWSER_PUBLIC_BASE_URL } else { (($env:GEN_CODE_BROWSER_PUBLIC_TARGETS -split ",")[0]).Trim() }
$env:GOTOOLCHAIN = if ($env:GOTOOLCHAIN) { $env:GOTOOLCHAIN } else { "auto" }
$env:PYTHONIOENCODING = if ($env:PYTHONIOENCODING) { $env:PYTHONIOENCODING } else { "utf-8" }
$scriptPath = Join-Path $projectRoot "scripts\verify-desktop-live-refresh.py"
$baselineLane = "canonical remote browser acceptance with desktop copy/runtime checks, richer authenticated fixture lane, and multi-site public-web read-only lanes (5174 + 10008)"
$uiBaseUri = [System.Uri]$env:GEN_CODE_UI_BASE_URL
$uiHost = $uiBaseUri.Host
$uiPort = if ($uiBaseUri.IsDefaultPort) { if ($uiBaseUri.Scheme -eq "https") { 443 } else { 80 } } else { $uiBaseUri.Port }

Write-Host "Desktop live refresh and copy/runtime alignment check"
Write-Host "  project root : $projectRoot"
Write-Host "  lane         : $baselineLane"
Write-Host "  script       : $scriptPath"
Write-Host "  UI base URL  : $env:GEN_CODE_UI_BASE_URL"
Write-Host "  API base URL : $env:GEN_CODE_API_BASE_URL"
Write-Host "  mode         : $env:GEN_CODE_ACCEPTANCE_MODE"
Write-Host "  browser-only : $env:GEN_CODE_BROWSER_ONLY_ACCEPTANCE"
Write-Host "  public web   : mode=$env:GEN_CODE_BROWSER_PUBLIC_WEB_MODE targets=$env:GEN_CODE_BROWSER_PUBLIC_TARGETS"
Write-Host "  failures     : remote canonical live matrix + browser navigation/richer-authenticated/multi-site-public-web lanes + fallback evidence-only"

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
$browserPolicyPath = Join-Path $artifactRoot "browser-policy.json"
$frontendStdout = Join-Path $projectRoot "desktop\frontend\vite-full.stdout.log"
$frontendStderr = Join-Path $projectRoot "desktop\frontend\vite-full.stderr.log"
$serverStdout = Join-Path $projectRoot "server-full.stdout.log"
$serverStderr = Join-Path $projectRoot "server-full.stderr.log"
$forceCurrentBootstrap = if ($env:GEN_CODE_FORCE_CURRENT_BOOTSTRAP) { $env:GEN_CODE_FORCE_CURRENT_BOOTSTRAP -ne "0" } else { $false }

if (-not (Test-Path $artifactRoot)) {
  New-Item -ItemType Directory -Path $artifactRoot | Out-Null
}
$env:GEN_CODE_ARTIFACT_DIR = $artifactRoot
$publicBrowserMode = $env:GEN_CODE_BROWSER_PUBLIC_WEB_MODE.Trim().ToLowerInvariant()
$publicBrowserSkipModes = @("skip", "disabled", "off", "false", "0")
$publicBrowserTargets = @()
$publicBrowserHosts = @()
if ($env:GEN_CODE_BROWSER_PUBLIC_TARGETS) {
  $publicBrowserTargets = @(
    $env:GEN_CODE_BROWSER_PUBLIC_TARGETS -split "[,\r\n]" |
      Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
      ForEach-Object { $_.Trim() }
  )
}
if ($publicBrowserTargets.Count -eq 0 -and -not [string]::IsNullOrWhiteSpace($env:GEN_CODE_BROWSER_PUBLIC_BASE_URL)) {
  $publicBrowserTargets = @($env:GEN_CODE_BROWSER_PUBLIC_BASE_URL.Trim())
}
if ($publicBrowserSkipModes -notcontains $publicBrowserMode) {
  if ($publicBrowserTargets.Count -eq 0) {
    Write-Error "Setup failure: public-web lane requires at least one explicit HTTPS target"
    exit 2
  }
  foreach ($target in $publicBrowserTargets) {
    try {
      $publicBrowserUri = [System.Uri]$target
      if ($publicBrowserUri.Scheme -ne "https" -or [string]::IsNullOrWhiteSpace($publicBrowserUri.Host)) {
        throw "public-web lane requires HTTPS targets"
      }
      $publicBrowserHosts += $publicBrowserUri.Host
    } catch {
      Write-Error "Setup failure: invalid public-web target '$target'. $_"
      exit 2
    }
  }
}
$existingAllowedHosts = @()
if ($env:GENCODE_BROWSER_ALLOWED_HOSTS) {
  $existingAllowedHosts += ($env:GENCODE_BROWSER_ALLOWED_HOSTS -split ",")
}
if ($env:GEN_CODE_BROWSER_ALLOWED_HOSTS) {
  $existingAllowedHosts += ($env:GEN_CODE_BROWSER_ALLOWED_HOSTS -split ",")
}
if ($publicBrowserHosts.Count -gt 0) {
  $existingAllowedHosts += $publicBrowserHosts
}
$normalizedAllowedHosts = @(
  $existingAllowedHosts |
    Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
    ForEach-Object { $_.Trim() } |
    Select-Object -Unique
)
$allowedHostsValue = ($normalizedAllowedHosts -join ",")
$env:GENCODE_BROWSER_ALLOWED_HOSTS = $allowedHostsValue
$env:GEN_CODE_BROWSER_ALLOWED_HOSTS = $allowedHostsValue
$browserPolicy = @{
  allowedHosts = $normalizedAllowedHosts
  hosts = @{
    "127.0.0.1" = @{
      sessionRequired = $true
      cookies = @(
        @{
          name = "gc_auth"
          value = "acceptance-session"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_profile"
          value = "acceptance"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_role"
          value = "reader"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_scope"
          value = "controlled"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_transport"
          value = "cookie"
          path = "/"
          sameSite = "Lax"
        }
      )
    }
    "localhost" = @{
      sessionRequired = $true
      cookies = @(
        @{
          name = "gc_auth"
          value = "acceptance-session"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_profile"
          value = "acceptance"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_role"
          value = "reader"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_scope"
          value = "controlled"
          path = "/"
          sameSite = "Lax"
        }
        @{
          name = "gc_auth_transport"
          value = "cookie"
          path = "/"
          sameSite = "Lax"
        }
      )
    }
  }
}
$browserPolicy | ConvertTo-Json -Depth 8 | Set-Content -Path $browserPolicyPath -Encoding UTF8
$env:GENCODE_BROWSER_POLICY_FILE = $browserPolicyPath
$env:GEN_CODE_BROWSER_POLICY_FILE = $browserPolicyPath
Write-Host "  browser cfg  : policy=$browserPolicyPath allowedHosts=$allowedHostsValue targets=$($publicBrowserTargets -join ',')"

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

function Get-RuntimeStatusSnapshot {
  param(
    [Parameter(Mandatory = $true)][string]$ApiBaseUrl
  )

  try {
    $payload = Invoke-WebRequest -UseBasicParsing "$ApiBaseUrl/api/runtime/status" -TimeoutSec 5
    $json = $payload.Content | ConvertFrom-Json
    return $json.data
  } catch {
    return $null
  }
}

function Test-BrowserCoreReady {
  param(
    [Parameter(Mandatory = $true)][string]$ApiBaseUrl,
    [Parameter(Mandatory = $true)][string]$UiBaseUrl
  )

  $threadName = "browser-preflight-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
  try {
    $threadPayload = @{ name = $threadName; permissionMode = "ask-user" } | ConvertTo-Json -Compress
    $threadResponse = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/api/threads" -ContentType "application/json" -Body $threadPayload -TimeoutSec 15
    $threadID = [string]$threadResponse.data.id
    if ([string]::IsNullOrWhiteSpace($threadID)) {
      return @{
        Ready = $false
        Reason = "browser preflight could not create a thread"
      }
    }

    $taskPayload = @{
      title = "Browser preflight open"
      kind = "browser.open"
      input = (@{ url = $UiBaseUrl } | ConvertTo-Json -Compress)
    } | ConvertTo-Json -Compress
    $taskResponse = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/api/threads/$threadID/tasks" -ContentType "application/json" -Body $taskPayload -TimeoutSec 15
    $taskID = [string]$taskResponse.data.id
    if ([string]::IsNullOrWhiteSpace($taskID)) {
      return @{
        Ready = $false
        Reason = "browser preflight could not create a browser.open task"
      }
    }

    $runResponse = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/api/threads/$threadID/tasks/$taskID/run" -ContentType "application/json" -Body "{}" -TimeoutSec 45
    $task = $runResponse.data
    $status = [string]$task.status
    $summary = [string]$task.resultSummary
    return @{
      Ready = ($status -eq "completed" -and $summary -like "browser tab opened:*")
      Status = $status
      Summary = $summary
      ThreadId = $threadID
      TaskId = $taskID
      Reason = if ($status -eq "completed" -and $summary -like "browser tab opened:*") { "" } else { "browser preflight did not complete successfully" }
    }
  } catch {
    return @{
      Ready = $false
      Reason = $_.Exception.Message
    }
  }
}

function Test-DesktopUiIdentity {
  param(
    [Parameter(Mandatory = $true)][string]$UiBaseUrl
  )

  try {
    $response = Invoke-WebRequest -UseBasicParsing $UiBaseUrl -TimeoutSec 5
    $content = [string]$response.Content
    return $content.IndexOf("<title>Gen Code Desktop</title>", [System.StringComparison]::OrdinalIgnoreCase) -ge 0
  } catch {
    return $false
  }
}

function Get-ListeningProcessInfo {
  param(
    [Parameter(Mandatory = $true)][int]$Port
  )

  try {
    $connection = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction Stop | Select-Object -First 1
    if ($null -eq $connection) {
      return $null
    }
    $processInfo = Get-CimInstance Win32_Process -Filter "ProcessId = $($connection.OwningProcess)" -ErrorAction SilentlyContinue
    return @{
      Port = $Port
      ProcessId = $connection.OwningProcess
      Name = if ($null -ne $processInfo) { $processInfo.Name } else { "" }
      CommandLine = if ($null -ne $processInfo) { $processInfo.CommandLine } else { "" }
    }
  } catch {
    return $null
  }
}

function Wait-PortReleased {
  param(
    [Parameter(Mandatory = $true)][int]$Port,
    [int]$TimeoutSeconds = 15
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $listener = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction Stop | Select-Object -First 1
      if ($null -eq $listener) {
        return $true
      }
    } catch {
      return $true
    }
    Start-Sleep -Milliseconds 500
  }
  return $false
}

function Prepare-ProjectProcessOnPort {
  param(
    [Parameter(Mandatory = $true)][int]$Port,
    [Parameter(Mandatory = $true)][string]$Label,
    [Parameter(Mandatory = $true)][string[]]$AllowedCommandFragments,
    [string]$ExpectedProjectRoot = ""
  )

  $processInfo = Get-ListeningProcessInfo -Port $Port
  if ($null -eq $processInfo) {
    return @{
      Found = $false
      Reuse = $false
      StartNew = $true
    }
  }

  $commandLine = [string]($processInfo.CommandLine)
  $isProjectOwned = $false
  foreach ($fragment in $AllowedCommandFragments) {
    if (-not [string]::IsNullOrWhiteSpace($fragment) -and $commandLine.IndexOf($fragment, [System.StringComparison]::OrdinalIgnoreCase) -ge 0) {
      $isProjectOwned = $true
      break
    }
  }

  if (-not $isProjectOwned -and $Port -eq 10008 -and -not [string]::IsNullOrWhiteSpace($ExpectedProjectRoot)) {
    $runtimeStatus = Get-RuntimeStatusSnapshot -ApiBaseUrl $env:GEN_CODE_API_BASE_URL
    if ($null -ne $runtimeStatus) {
      $runtimeProjectRoot = [string]($runtimeStatus.projectRoot)
      if ($runtimeProjectRoot.Equals($ExpectedProjectRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
        $isProjectOwned = $true
      }
    }
  }

  if (-not $isProjectOwned -and $Port -eq $uiPort) {
    if (Test-DesktopUiIdentity -UiBaseUrl $env:GEN_CODE_UI_BASE_URL) {
      $isProjectOwned = $true
    }
  }

  if (-not $isProjectOwned) {
    throw "Port $Port is occupied by a non-project process (pid=$($processInfo.ProcessId), name=$($processInfo.Name), commandLine=$commandLine). Clear the port or set GEN_CODE_FORCE_CURRENT_BOOTSTRAP=0 to reuse the existing environment."
  }

  try {
    Write-Host "  bootstrap    : stopping existing $Label process on port $Port (pid=$($processInfo.ProcessId))"
    Stop-Process -Id $processInfo.ProcessId -Force
    if (-not (Wait-PortReleased -Port $Port -TimeoutSeconds 15)) {
      throw "port $Port did not release after stopping pid=$($processInfo.ProcessId)"
    }
    return @{
      Found = $true
      Reuse = $false
      StartNew = $true
    }
  } catch {
    Write-Host "  bootstrap    : reusing existing $Label process on port $Port because it already belongs to this repo and could not be stopped cleanly"
    return @{
      Found = $true
      Reuse = $true
      StartNew = $false
    }
  }
}

$serverProcess = $null
$frontendProcess = $null
$bootstrapStarted = $false
$startedServer = $false
$startedFrontend = $false
$usedExistingCanonicalInstance = $false

function Stop-BootstrapProcesses {
  param(
    $FrontendProcess,
    $ServerProcess
  )

  foreach ($proc in @($FrontendProcess, $ServerProcess)) {
    if ($null -ne $proc) {
      try {
        $targetId = $proc.Id
        Stop-Process -Id $targetId -Force -ErrorAction SilentlyContinue
        Wait-Process -Id $targetId -Timeout 15 -ErrorAction SilentlyContinue
      } catch {
      }
    }
  }
}

try {
  $apiReady = Wait-HttpOk -Url "$($env:GEN_CODE_API_BASE_URL)/api/runtime/status" -TimeoutSeconds 5
  $uiReady = Wait-HttpOk -Url $env:GEN_CODE_UI_BASE_URL -TimeoutSeconds 5
  $baseline = if ($apiReady) { Test-MCPBaselineReady -ApiBaseUrl $env:GEN_CODE_API_BASE_URL } else { @{ Ready = $false; Missing = @("external-fixture", "sdk-external-fixture", "third-party-time"); ServerIds = @() } }
  $browserPreflight = if ($apiReady -and $uiReady -and $baseline.Ready) { Test-BrowserCoreReady -ApiBaseUrl $env:GEN_CODE_API_BASE_URL -UiBaseUrl $env:GEN_CODE_UI_BASE_URL } else { @{ Ready = $false; Reason = "skipped until API, UI, and MCP baseline are ready" } }

  if ($forceCurrentBootstrap) {
    Write-Host "  bootstrap    : forcing current repo bootstrap for full acceptance"
    if ($apiReady) {
      Write-Host ("  existing API : reachable on 10008" + $(if ($baseline.Ready) { " with complete MCP baseline" } else { " but MCP baseline is incomplete: " + ($baseline.Missing -join ", ") }))
    }
    if ($uiReady) {
      Write-Host "  existing UI  : reachable on $uiPort"
    }
    if ($browserPreflight.Ready) {
      Write-Host "  browser gate : existing browser preflight is healthy"
    } else {
      Write-Host "  browser gate : existing browser preflight is not healthy, forcing fresh bootstrap"
      if ($browserPreflight.Reason) {
        Write-Host "  browser note : $($browserPreflight.Reason)"
      }
    }

    $serverDecision = Prepare-ProjectProcessOnPort -Port 10008 -Label "server" -AllowedCommandFragments @(
      "go run .\cmd\server",
      "go.exe run .\cmd\server",
      "gen-code"
    ) -ExpectedProjectRoot $projectRoot
    $frontendDecision = Prepare-ProjectProcessOnPort -Port $uiPort -Label "frontend" -AllowedCommandFragments @(
      "npm run dev",
      "vite",
      $frontendRoot,
      "gen-code"
    )

    if ($serverDecision.StartNew) {
      $serverProcess = Start-Process -FilePath $goCommand.Source `
        -ArgumentList @("run", ".\cmd\server") `
        -WorkingDirectory $projectRoot `
        -RedirectStandardOutput $serverStdout `
        -RedirectStandardError $serverStderr `
        -WindowStyle Hidden `
        -PassThru
      $startedServer = $true
    }

    if ($frontendDecision.StartNew) {
      $frontendProcess = Start-Process -FilePath $npmCommand.Source `
        -ArgumentList @("run", "dev", "--", "--host", $uiHost, "--port", [string]$uiPort) `
        -WorkingDirectory $frontendRoot `
        -RedirectStandardOutput $frontendStdout `
        -RedirectStandardError $frontendStderr `
        -WindowStyle Hidden `
        -PassThru
      $startedFrontend = $true
    }

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
    $browserPreflight = Test-BrowserCoreReady -ApiBaseUrl $env:GEN_CODE_API_BASE_URL -UiBaseUrl $env:GEN_CODE_UI_BASE_URL
    if (-not $browserPreflight.Ready) {
      throw ("Bootstrapped canonical instance failed browser preflight: " + $browserPreflight.Reason + $(if ($browserPreflight.Summary) { " / " + $browserPreflight.Summary } else { "" }))
    }
    $bootstrapStarted = ($startedServer -or $startedFrontend)
  } elseif (-not ($apiReady -and $uiReady -and $baseline.Ready -and $browserPreflight.Ready)) {
    Write-Host "  bootstrap    : starting current-code server/frontend because existing canonical instance is missing or incomplete"
    if ($apiReady -and -not $baseline.Ready) {
      Write-Host ("  baseline     : existing API instance missing MCP lanes: " + ($baseline.Missing -join ", "))
    }
    if ($apiReady -and $uiReady -and $baseline.Ready -and -not $browserPreflight.Ready) {
      Write-Host ("  browser gate : existing canonical browser preflight failed: " + $browserPreflight.Reason + $(if ($browserPreflight.Summary) { " / " + $browserPreflight.Summary } else { "" }))
    }

    $serverDecision = Prepare-ProjectProcessOnPort -Port 10008 -Label "server" -AllowedCommandFragments @(
      "go run .\cmd\server",
      "go.exe run .\cmd\server",
      "gen-code"
    ) -ExpectedProjectRoot $projectRoot
    $frontendDecision = Prepare-ProjectProcessOnPort -Port $uiPort -Label "frontend" -AllowedCommandFragments @(
      "npm run dev",
      "vite",
      $frontendRoot,
      "gen-code"
    )

    if ($serverDecision.StartNew) {
      $serverProcess = Start-Process -FilePath $goCommand.Source `
        -ArgumentList @("run", ".\cmd\server") `
        -WorkingDirectory $projectRoot `
        -RedirectStandardOutput $serverStdout `
        -RedirectStandardError $serverStderr `
        -WindowStyle Hidden `
        -PassThru
      $startedServer = $true
    }

    if ($frontendDecision.StartNew) {
      $frontendProcess = Start-Process -FilePath $npmCommand.Source `
        -ArgumentList @("run", "dev", "--", "--host", $uiHost, "--port", [string]$uiPort) `
        -WorkingDirectory $frontendRoot `
        -RedirectStandardOutput $frontendStdout `
        -RedirectStandardError $frontendStderr `
        -WindowStyle Hidden `
        -PassThru
      $startedFrontend = $true
    }

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
    $browserPreflight = Test-BrowserCoreReady -ApiBaseUrl $env:GEN_CODE_API_BASE_URL -UiBaseUrl $env:GEN_CODE_UI_BASE_URL
    if (-not $browserPreflight.Ready) {
      throw ("Bootstrapped canonical instance failed browser preflight: " + $browserPreflight.Reason + $(if ($browserPreflight.Summary) { " / " + $browserPreflight.Summary } else { "" }))
    }
    $bootstrapStarted = ($startedServer -or $startedFrontend)
  } else {
    Write-Host "  bootstrap    : using existing canonical instance with complete MCP baseline and healthy browser preflight"
    $usedExistingCanonicalInstance = $true
  }

  & $pythonCommand.Source $scriptPath
  $exitCode = $LASTEXITCODE
  if ($exitCode -ne 0 -and $usedExistingCanonicalInstance -and -not $forceCurrentBootstrap) {
    Write-Host "  retry        : existing canonical instance failed during verification, retrying with current repo bootstrap"
    $env:GEN_CODE_FORCE_CURRENT_BOOTSTRAP = "1"
    if ($bootstrapStarted) {
      Stop-BootstrapProcesses -FrontendProcess $frontendProcess -ServerProcess $serverProcess
      $bootstrapStarted = $false
    }
    & powershell -ExecutionPolicy Bypass -File $PSCommandPath
    $exitCode = $LASTEXITCODE
  }
  if ($exitCode -ne 0) {
    Write-Error "Verification failed with exit code $exitCode"
    exit $exitCode
  }
} finally {
  if ($bootstrapStarted) {
    Stop-BootstrapProcesses -FrontendProcess $frontendProcess -ServerProcess $serverProcess
  }
}
