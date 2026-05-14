# Desktop

This directory contains the standalone Wails desktop shell for the project.

## Requirements

- Go installed and available in PATH
- Wails CLI installed:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

If `wails` is not in PATH after install, add your Go bin directory to PATH.

## Development

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
wails dev
```

## Build

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
wails build
```
