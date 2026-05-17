$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

$env:GEN_CODE_UI_BASE_URL = if ($env:GEN_CODE_UI_BASE_URL) { $env:GEN_CODE_UI_BASE_URL } else { "http://127.0.0.1:5174/" }
$env:GEN_CODE_API_BASE_URL = if ($env:GEN_CODE_API_BASE_URL) { $env:GEN_CODE_API_BASE_URL } else { "http://127.0.0.1:10008" }

python .\scripts\verify-desktop-live-refresh.py
