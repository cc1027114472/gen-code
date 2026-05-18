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

当前环境现状：

- 本机已安装 `gh`
- 已登录 GitHub 账号 `cc1027114472`
- 当前仓库已绑定 GitHub 远端 `https://github.com/cc1027114472/gen-code.git`

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

## 失败分类基线

首次真实远端运行失败时，优先按 `desktop-smoke-failure.json` 中的 `category` 判断：

- `api-unavailable`
  - runtime API 未启动、未监听、启动过慢，或 runner 无法连到 `10008`
- `page-load-failed`
  - 前端页面未成功加载到可用工作台状态，或线程卡片始终未渲染
- `runtime-lane-assertion`
  - 页面已打开，但 `remote-app-server / canonical`、`canonicalRuntimeUrl` 等运行链路语义不满足预期
- `refresh-mode-assertion`
  - 页面刷新方式与预期不一致，例如 smoke 预期的 SSE 连接态没有正确显示
- `desktop-copy-assertion`
  - 首屏文案断言失败，或出现了明确不应存在的陈旧文案
- `unknown`
  - 仍未归类的异常；优先结合 screenshot 和 stderr 继续排查

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

## 首次真实远端运行记录

首次远端首跑完成后，在验收报告中补一段固定记录，至少包括：

- run 日期
- branch / ref
- workflow：`desktop-smoke.yml`
- run id
- 结论：success / failed
- `desktop-smoke-summary` artifact 是否可下载
- 若失败：
  - `desktop-smoke-failure.json` 中的 `category`
  - 首个有效定位证据
  - 是否属于 CI-only 问题

当前真实状态：

- 本机已安装 `gh`
- 已完成 `gh auth login`
- 当前仓库已绑定 GitHub 远端
- 已完成一次真实远端 `desktop-smoke.yml` 首跑

## 首次真实远端运行记录

- 运行日期：2026-05-18
- branch / ref：`master`
- workflow：`desktop-smoke.yml`
- run id：`26040617047`
- run URL：[desktop-smoke #26040617047](https://github.com/cc1027114472/gen-code/actions/runs/26040617047)
- 结论：`success`
- `desktop-smoke-summary` artifact：可下载，已核对
- summary 关键字段：
  - `acceptanceMode=smoke`
  - `runtimeSource=remote-app-server`
  - `runtimeTrust=canonical`
  - `canonicalRuntimeUrl=http://127.0.0.1:10008`
  - `refreshMode.label=SSE 已连接`
  - `threadId=thread-1`
- 首个有效证据：
  - 远端 artifact `desktop-smoke-summary.json`
  - 本地下载核对路径：`tmp/gh-run-26040617047/desktop-smoke-summary.json`
- CI-only 问题：本次未触发功能性 CI-only 失败
- 额外观察：
  - GitHub Actions 给出 Node.js 20 deprecation annotation，当前不影响通过
  - `windows-latest` 正在向 `windows-2025-vs2026` 重定向，当前不影响通过

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
- summary 中 `runtimeSource=remote-app-server`
- summary 中 `runtimeTrust=canonical`
- summary 中 `refreshMode` 不再是无意义的空值或 `unknown`
- 无需依赖人工解释即可判断 smoke gate 是否通过
