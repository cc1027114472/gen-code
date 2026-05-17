$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot
$env:GOTOOLCHAIN = if ($env:GOTOOLCHAIN) { $env:GOTOOLCHAIN } else { "auto" }
go run .\cmd\server
