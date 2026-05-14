$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot
go run .\cmd\server
