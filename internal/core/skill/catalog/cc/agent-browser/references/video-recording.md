# 视频录制

把浏览器自动化过程录制成视频，用于调试、文档记录或验证。

**相关内容：** 完整命令参考见 [commands.md](commands.md)，快速开始见 [SKILL.md](../SKILL.md)。

## 目录

- [基础录制](#基础录制)
- [录制命令](#录制命令)
- [使用场景](#使用场景)
- [最佳实践](#最佳实践)
- [输出格式](#输出格式)
- [限制](#限制)

## 基础录制

```bash
# 开始录制
agent-browser record start ./demo.webm

# 执行动作
agent-browser open https://example.com
agent-browser snapshot -i
agent-browser click @e1
agent-browser fill @e2 "test input"

# 停止并保存
agent-browser record stop
```

## 录制命令

```bash
# 开始录制到文件
agent-browser record start ./output.webm

# 停止当前录制
agent-browser record stop

# 使用新文件重新开始录制（会先停止当前录制）
agent-browser record restart ./take2.webm
```

## 使用场景

### 调试失败的自动化

```bash
#!/bin/bash
# 为调试录制自动化过程

agent-browser record start ./debug-$(date +%Y%m%d-%H%M%S).webm

# 运行你的自动化
agent-browser open https://app.example.com
agent-browser snapshot -i
agent-browser click @e1 || {
    echo "点击失败 - 请检查录制视频"
    agent-browser record stop
    exit 1
}

agent-browser record stop
```

### 生成文档材料

```bash
#!/bin/bash
# 为文档录制工作流

agent-browser record start ./docs/how-to-login.webm

agent-browser open https://app.example.com/login
agent-browser wait 1000  # 为了可见性暂停

agent-browser snapshot -i
agent-browser fill @e1 "demo@example.com"
agent-browser wait 500

agent-browser fill @e2 "password"
agent-browser wait 500

agent-browser click @e3
agent-browser wait --load networkidle
agent-browser wait 1000  # 展示结果

agent-browser record stop
```

### CI/CD 测试证据

```bash
#!/bin/bash
# 为 CI 产物录制 E2E 测试过程

TEST_NAME="${1:-e2e-test}"
RECORDING_DIR="./test-recordings"
mkdir -p "$RECORDING_DIR"

agent-browser record start "$RECORDING_DIR/$TEST_NAME-$(date +%s).webm"

# 运行测试
if run_e2e_test; then
    echo "测试通过"
else
    echo "测试失败 - 录制已保存"
fi

agent-browser record stop
```

## 最佳实践

### 1. 适当加入暂停，提升可读性

```bash
# 为人工回看放慢一点
agent-browser click @e1
agent-browser wait 500  # 让观看者看到结果
```

### 2. 使用有语义的文件名

```bash
# 在文件名中带上上下文
agent-browser record start ./recordings/login-flow-2024-01-15.webm
agent-browser record start ./recordings/checkout-test-run-42.webm
```

### 3. 在错误场景下也正确处理录制

```bash
#!/bin/bash
set -e

cleanup() {
    agent-browser record stop 2>/dev/null || true
    agent-browser close 2>/dev/null || true
}
trap cleanup EXIT

agent-browser record start ./automation.webm
# ... 自动化步骤 ...
```

### 4. 与截图配合使用

```bash
# 录视频，同时保留关键帧截图
agent-browser record start ./flow.webm

agent-browser open https://example.com
agent-browser screenshot ./screenshots/step1-homepage.png

agent-browser click @e1
agent-browser screenshot ./screenshots/step2-after-click.png

agent-browser record stop
```

## 输出格式

- 默认格式：WebM（VP8/VP9 编码）
- 兼容所有现代浏览器和视频播放器
- 压缩率高，同时保持较好画质

## 限制

- 录制会给自动化带来少量额外开销
- 较大的录制文件可能占用较多磁盘空间
- 某些 headless 环境可能存在编解码器限制
