---
name: go-backend-clean-architecture
description: 使用 Gin、MongoDB、JWT 认证和整洁架构的 Go 后端。
---

# Go 整洁架构

一个使用 Gin、MongoDB、JWT 身份验证和 Docker 支持的 Go 后端，遵循整洁架构原则。

## 技术栈

- **框架**：Gin
- **语言**：Go
- **数据库**：MongoDB
- **认证**：JWT
- **架构**：Clean Architecture

## 前置条件

- Go 1.21 或更高版本
- 已安装并可访问的 MongoDB
- Docker（可选）

## 设置

### 1. 克隆模板

```bash
git clone --depth 1 https://github.com/amitshekhariitbhu/go-backend-clean-architecture.git .
```

如果目录不是空的：

```bash
git clone --depth 1 https://github.com/amitshekhariitbhu/go-backend-clean-architecture.git _temp_template
mv _temp_template/* _temp_template/.* . 2>/dev/null || true
rm -rf _temp_template
```

### 2. 删除 Git 历史（可选）

```bash
rm -rf .git
git init
```

### 3. 安装依赖

```bash
go mod download
```

### 4. 设置环境

配置 MongoDB 连接和 JWT 密钥。

## 构建

```bash
go build -o app ./cmd/main.go
```

## 开发

```bash
go run ./cmd/main.go
```
