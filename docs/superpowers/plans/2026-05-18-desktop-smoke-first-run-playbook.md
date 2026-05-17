# Desktop Smoke Gate 首次真实运行手册

## 当前前提

本仓库已经具备以下内容：

- 默认 smoke gate workflow：
  - [.github/workflows/desktop-smoke.yml](/D:/GOWorks/gen-code-heji/gen-code/.github/workflows/desktop-smoke.yml)
- 本地对等入口：
  - [run-desktop-smoke-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-check.ps1)
  - [run-desktop-smoke-with-bootstrap.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-with-bootstrap.ps1)
- smoke summary / failure artifact 逻辑：
  - [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)

当前环境限制：

- 本机已安装 `gh`
- 但当前 **未登录 GitHub**
- 因此本轮不能伪装成“已经触发了远端 workflow_dispatch”

## 首次真实运行步骤

### 1. 登录 GitHub CLI

在仓库根目录或任意终端执行：

```powershell
gh auth login
```

完成后验证：

```powershell
gh auth status
```

预期：能看到已登录的 GitHub host 和账号。

### 2. 确认仓库远端

执行：

```powershell
git remote -v
```

预期：当前仓库已绑定到真实 GitHub 远端。

### 3. 手动触发 smoke gate

执行：

```powershell
gh workflow run desktop-smoke.yml
```

如果需要指定分支：

```powershell
gh workflow run desktop-smoke.yml --ref <branch>
```

### 4. 查看最近一次运行

执行：

```powershell
gh run list --workflow desktop-smoke.yml --limit 5
```

然后查看详情：

```powershell
gh run view <run-id>
```

若要流式跟踪日志：

```powershell
gh run watch <run-id>
```

## 预期产物

### 成功时

应能看到 artifact：

- `desktop-smoke-summary`

其中至少应包含：

- `desktop-smoke-summary.json`

重点检查字段：

- `acceptanceMode=smoke`
- `runtimeSource=remote-app-server`
- `runtimeTrust=canonical`
- `refreshMode.label`
- `copyAndRuntimeConsistency.checkedTexts`

### 失败时

应能看到 artifact：

- `desktop-smoke-logs`
- `desktop-smoke-summary`
- `desktop-smoke-screenshot`

重点文件：

- `server-smoke.stdout.log`
- `server-smoke.stderr.log`
- `desktop/frontend/vite-smoke.stdout.log`
- `desktop/frontend/vite-smoke.stderr.log`
- `desktop-smoke-failure.json`
- `desktop-smoke-failure.png`

## 首次失败时的排查顺序

1. 先看 `desktop-smoke-failure.json`
2. 再看 `desktop-smoke-failure.png`
3. 再对照：
   - `server-smoke.stderr.log`
   - `vite-smoke.stderr.log`
4. 最后再回看 workflow step 日志

推荐判断顺序：

- 如果 server stderr 有异常：优先看 API 是否启动失败
- 如果 vite stderr 有异常：优先看前端 dev server 是否启动失败
- 如果 screenshot 显示页面已打开但断言失败：优先看 smoke 文案或 runtime 展示是否回归

## 与本地对等入口的关系

真实 workflow 之前，建议先执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-with-bootstrap.ps1
```

它是当前 smoke gate 的本地对等路径。若这条都不通过，远端 workflow 大概率也不会通过。

## 通过标准

首次真实运行通过时，至少应满足：

- workflow 成功
- `desktop-smoke-summary` artifact 可下载
- summary 中 `refreshMode` 不再是无意义的空值
- 无需依赖人工解释即可判断 smoke gate 是否通过
