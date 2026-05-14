# Desktop Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在仓库中新增一个独立可运行的 `desktop/` Wails 桌面端子项目，打通窗口启动、前端渲染和 Go 绑定调用。

**Architecture:** 桌面端作为与现有 Go 服务解耦的子项目存在，目录落在 `desktop/` 下。Wails 负责桌面窗口和前后端桥接，Go 侧暴露最小 `App` 绑定对象，前端首页只验证桌面端启动和绑定调用是否成功。

**Tech Stack:** Go, Wails v2, Vite, React, TypeScript, PowerShell

---

## File Structure

- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\main.go`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\app.go`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\app_test.go`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\go.mod`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\wails.json`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\README.md`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\package.json`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\tsconfig.json`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\vite.config.ts`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\index.html`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\main.tsx`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\style.css`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\.gitignore`

### Task 1: Prepare Wails Tooling And Project Boundary

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\.gitignore`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\README.md`

- [ ] **Step 1: Verify the current Go toolchain and check whether Wails CLI is installed**

Run:

```powershell
go version
wails version
```

Expected:

- `go version` prints the installed Go version
- `wails version` either prints the version or fails with `CommandNotFoundException`

- [ ] **Step 2: Install Wails CLI if it is missing**

Run:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Expected:

- The command completes without error
- The binary is installed into the Go bin directory

- [ ] **Step 3: Confirm Wails can run and document the requirement**

Add this content to `D:\GOWorks\gen-code-heji\gen-code\desktop\README.md`:

```md
# Desktop

This directory contains the standalone Wails desktop shell for the project.

## Requirements

- Go installed and available in PATH
- Wails CLI installed:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

If `wails` is not in PATH after install, add your Go bin directory to PATH.
```

- [ ] **Step 4: Ignore Wails build output and generated frontend artifacts**

Update `D:\GOWorks\gen-code-heji\gen-code\.gitignore` to include:

```gitignore
desktop/build/
desktop/frontend/dist/
desktop/frontend/node_modules/
```

- [ ] **Step 5: Re-open the repository tree to verify the desktop boundary is isolated**

Run:

```powershell
Get-ChildItem D:\GOWorks\gen-code-heji\gen-code
```

Expected:

- Existing server files remain untouched
- `desktop/` will be the only new top-level app directory for the desktop shell

### Task 2: Create The Minimal Go Desktop App

**Files:**
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\go.mod`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\main.go`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\app.go`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\app_test.go`

- [ ] **Step 1: Write the failing Go test for the desktop binding**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\app_test.go`:

```go
package main

import "testing"

func TestAppGetAppInfo(t *testing.T) {
	app := NewApp()

	if got := app.GetAppInfo(); got != "gen-code desktop shell ready" {
		t.Fatalf("GetAppInfo() = %q, want %q", got, "gen-code desktop shell ready")
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

Run:

```powershell
go test .\desktop
```

Expected:

- FAIL because `NewApp` or `GetAppInfo` is not defined yet

- [ ] **Step 3: Add the desktop Go module**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\go.mod`:

```go
module gen-code/desktop

go 1.24.0

require github.com/wailsapp/wails/v2 v2.10.1
```

- [ ] **Step 4: Implement the minimal app binding**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\app.go`:

```go
package main

import "context"

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAppInfo() string {
	return "gen-code desktop shell ready"
}
```

- [ ] **Step 5: Implement the Wails entrypoint**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\main.go`:

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Gen Code Desktop",
		Width:  1200,
		Height: 760,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("error:", err.Error())
	}
}
```

- [ ] **Step 6: Run the Go test again to verify it passes**

Run:

```powershell
go test .\desktop
```

Expected:

- PASS

### Task 3: Create The Frontend Shell

**Files:**
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\package.json`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\tsconfig.json`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\vite.config.ts`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\index.html`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\main.tsx`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\style.css`

- [ ] **Step 1: Add the frontend package manifest**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\package.json`:

```json
{
  "name": "gen-code-desktop-frontend",
  "private": true,
  "version": "0.0.1",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@types/react": "^18.3.3",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.1",
    "typescript": "^5.6.3",
    "vite": "^5.4.10"
  }
}
```

- [ ] **Step 2: Add TypeScript and Vite config**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": false,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": false,
    "jsx": "react-jsx",
    "strict": true
  },
  "include": ["src"]
}
```

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\vite.config.ts`:

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
```

- [ ] **Step 3: Add the frontend HTML shell**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\index.html`:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Gen Code Desktop</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 4: Add the React entrypoint**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\main.tsx`:

```tsx
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./style.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
```

- [ ] **Step 5: Add the minimal desktop shell UI**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`:

```tsx
import { useState } from "react";

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetAppInfo: () => Promise<string>;
        };
      };
    };
  }
}

export default function App() {
  const [message, setMessage] = useState("Waiting for Go bridge...");
  const [error, setError] = useState("");

  const handleCheck = async () => {
    setError("");

    try {
      const result = await window.go?.main?.App?.GetAppInfo?.();
      setMessage(result ?? "Go bridge not available");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown bridge error");
    }
  };

  return (
    <main className="shell">
      <section className="panel">
        <p className="eyebrow">Wails Desktop Shell</p>
        <h1>Gen Code Desktop</h1>
        <p className="lead">
          The standalone desktop shell is running and ready for future business integration.
        </p>
        <button onClick={handleCheck}>Check Go Binding</button>
        <p className="message">{message}</p>
        {error ? <p className="error">{error}</p> : null}
      </section>
    </main>
  );
}
```

- [ ] **Step 6: Add minimal styling**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\style.css`:

```css
:root {
  color-scheme: light;
  font-family: "Segoe UI", "PingFang SC", sans-serif;
  background:
    radial-gradient(circle at top, #f0f7ff 0%, #f7f3ec 42%, #f5efe6 100%);
  color: #172033;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  min-height: 100vh;
}

#root {
  min-height: 100vh;
}

.shell {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 32px;
}

.panel {
  width: min(680px, 100%);
  padding: 40px;
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(23, 32, 51, 0.08);
  box-shadow: 0 24px 80px rgba(38, 56, 88, 0.12);
}

.eyebrow {
  margin: 0 0 12px;
  text-transform: uppercase;
  letter-spacing: 0.18em;
  font-size: 12px;
  color: #9a6c38;
}

h1 {
  margin: 0;
  font-size: 52px;
  line-height: 1.02;
}

.lead {
  margin: 18px 0 28px;
  font-size: 18px;
  line-height: 1.7;
  color: #49566f;
}

button {
  border: 0;
  border-radius: 999px;
  background: #172033;
  color: #fff;
  padding: 14px 22px;
  font-size: 16px;
  cursor: pointer;
}

.message,
.error {
  margin-top: 18px;
  font-size: 16px;
}

.error {
  color: #b42318;
}
```

- [ ] **Step 7: Install frontend dependencies and build the frontend**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm install
npm run build
```

Expected:

- `node_modules` is created
- `dist` is generated without build errors

### Task 4: Add Wails Config And Verify The Desktop App

**Files:**
- Create: `D:\GOWorks\gen-code-heji\gen-code\desktop\wails.json`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\README.md`

- [ ] **Step 1: Add the Wails project config**

Create `D:\GOWorks\gen-code-heji\gen-code\desktop\wails.json`:

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "gen-code-desktop",
  "outputfilename": "gen-code-desktop",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto",
  "frontenddir": "frontend"
}
```

- [ ] **Step 2: Expand the desktop README with run commands**

Append this content to `D:\GOWorks\gen-code-heji\gen-code\desktop\README.md`:

```md
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
```

- [ ] **Step 3: Run Wails code generation and desktop dev boot**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
wails dev
```

Expected:

- Wails installs frontend dependencies if needed
- The app window opens
- The home page renders
- Clicking `Check Go Binding` shows `gen-code desktop shell ready`

- [ ] **Step 4: Run the desktop Go test after wiring Wails**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
go test ./...
```

Expected:

- PASS

- [ ] **Step 5: Run the existing server project test suite to verify isolation**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./...
```

Expected:

- Existing server-side tests still pass
- The new desktop project does not break the current backend skeleton
