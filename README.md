<div align="center">

[![AI-Assisted Development](https://img.shields.io/badge/AI--Assisted-Development-blueviolet?style=flat-square&logo=openai)](https://github.com/kilolonion/cursor-cpa-plugin)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.24-00ADD8?style=flat-square&logo=go)](go.mod)

**🤖 AI-Assisted Development** · 本项目的自研补丁开发、测试验证与文档编写由 AI 辅助完成

</div>

# Cursor for CPA/CLIProxyAPI (Enhanced)

> 基于 [yobo2u/omsub](https://github.com/yobo2u/omsub) `cursor` 分支 0.6.1 的增强版本

这是一个独立的 CLIProxyAPI 原生动态库插件，把用户本人授权的 Cursor 订阅接入 OpenAI 兼容的 `/v1/chat/completions` 接口。相比上游版本，增加了**思考透传**、**纯文本约束注入**和**官方图标支持**三项自研功能。

> [!IMPORTANT]
> 本项目是非官方社区插件，与 Cursor、Anysphere、CLIProxyAPI、CPA Manager Plus 无隶属或授权关系。使用前请完整阅读[免责声明](DISCLAIMER.md)。

## 与上游的主要差异

本仓库在 0.6.1 基础上增加了以下增强：

### 1. 思考透传（Reasoning Content Streaming）

上游版本会丢弃 Cursor 的 `thinking_delta` 事件，客户端无法看到模型的思考过程。本版本：

- 流式响应中通过标准 `delta.reasoning_content` 字段实时透传思考内容
- 非流式响应中在 `message.reasoning_content` 中返回完整思考文本
- 与正文内容完全解耦，客户端可单独渲染思考块

**代码位置**：
- `internal/openai/response.go` — `StreamReasoning()` / `AddReasoning()` / `ReasoningContent` 字段
- `internal/plugin/executor.go` — 流式与非流式思考事件路由

### 2. 纯文本约束注入（System Constraint Injection）

当请求未声明任何工具时（`len(tools) == 0`），上游会触发 Cursor Composer 2.5 的默认工具链（如 `web_search_request_query`），但由于 CPA 没有工具回传能力，导致空转等待 90 秒超时，客户端只能收到碎片字符。

本版本在用户 prompt 前注入 `<system_constraint>` 约束指令，明确声明当前处于纯文本对话模式，避免模型尝试发起工具或搜索调用：

```text
<system_constraint>
NOTE: This is a direct conversational session with no tool execution environment.
Please answer directly in text using your knowledge without attempting to invoke tools or web search.
</system_constraint>
```

**代码位置**：`internal/cursorproto/request.go` — `buildAction()`

### 3. 官方图标（Official Brand Logo）

上游版本 `metadata.Logo` 字段为空，CPAMP 面板显示默认占位符。本版本：

- 使用 Cursor 官方品牌资源 `brand-logo-5.svg`
- 插件注册信息和账户元数据均携带图标 URL
- 面板正确显示 Cursor 官方立方体标志

**代码位置**：`internal/plugin/handler.go` — `pluginLogoURL` 常量与 `metadata.Logo` 字段

## 功能特性

- ✅ OpenAI Chat Completions 兼容接口
- ✅ SSE 流式响应（含思考流）
- ✅ 标准 function tools 与 `tool_choice`
- ✅ 多轮工具调用与结果续轮
- ✅ 会话检查点复用（追加式历史优化）
- ✅ 图片与文本附件
- ✅ Cursor 管理页（模型禁用、用量估算）
- ✅ OAuth 动态模型发现

## 安装

### 从 Release 安装

下载最新 Release 的 Linux amd64 包：

```sh
curl -LO https://github.com/kilolonion/cursor-cpa-plugin/releases/download/v0.6.1-enhanced/cursor_0.6.1-enhanced_linux_amd64.zip
unzip cursor_0.6.1-enhanced_linux_amd64.zip -d cursor-plugin
```

将 `cursor.so` 复制到 CLIProxyAPI 插件目录：

```sh
sudo mkdir -p /opt/cpa-manager-plus/cliproxyapi/plugins/linux/amd64
sudo cp cursor-plugin/cursor.so /opt/cpa-manager-plus/cliproxyapi/plugins/linux/amd64/
```

在 `config.yaml` 中启用插件：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cursor:
      enabled: true
      priority: 1
```

重启 CLIProxyAPI：

```sh
docker restart cli-proxy-api
```

### 从源码构建

需要 Go 1.24+ 和 CGO 工具链：

```sh
git clone https://github.com/kilolonion/cursor-cpa-plugin.git
cd cursor-cpa-plugin

CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o cursor.so .
```

## 调用示例

### 流式请求（含思考过程）

```bash
curl https://your-cpa.example/v1/chat/completions \
  -H "Authorization: Bearer $CPA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "cursor/composer-2-5",
    "messages": [{"role": "user", "content": "推导快排平均复杂度"}],
    "stream": true
  }'
```

响应中的思考内容：

```json
{"id":"chatcmpl-xxx","choices":[{"index":0,"delta":{"reasoning_content":"正在推导快排..."}}]}
```

### 非流式请求

```bash
curl https://your-cpa.example/v1/chat/completions \
  -H "Authorization: Bearer $CPA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "cursor/cursor-grok-4-6-high",
    "messages": [{"role": "user", "content": "hello"}],
    "stream": false
  }'
```

响应中的思考字段：

```json
{
  "choices": [{
    "message": {
      "content": "Hello! How can I help?",
      "reasoning_content": "用户发送了简单问候..."
    }
  }]
}
```

## 开发

### 运行测试

```sh
go test ./...
```

### 代码检查

```sh
gofmt -l .
go vet ./...
```

## 许可证

MIT License — 详见 [LICENSE](LICENSE)

**版权声明**：
- 原始代码：Copyright (c) 2026 Cursor for CPA/CLIProxyAPI contributors
- 自研补丁：Copyright (c) 2026 kilolonion

## 致谢与来源

- **上游仓库**：[yobo2u/omsub](https://github.com/yobo2u/omsub)（cursor 分支）
- **协议参考**：[opencodex](https://github.com/lidge-jun/opencodex)（MIT License）
- **ABI 参考**：[CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)（MIT License）
- **图标来源**：Cursor 官方品牌资源（[cursor.com/cn/brand](https://cursor.com/cn/brand)）

详见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)

## AI 辅助开发说明

本项目的自研补丁（思考透传、约束注入、图标集成）由 AI 辅助完成，包括：

- 需求分析与实现方案设计
- 代码编写与测试用例更新
- 端到端行为验证（与上游二进制对比）
- 文档撰写

人工审查与最终决策由 [kilolonion](https://github.com/kilolonion) 完成。

## 贡献

本仓库欢迎 PR 和 Issue。与上游相比，本仓库承诺：

- **及时响应**：48 小时内回复 Issue 和 PR
- **明确反馈**：拒绝时会说明理由，接受时会指明合并计划
- **维护活跃**：持续跟进 Cursor 协议变化与 CLIProxyAPI 更新

详见 [CONTRIBUTING.md](CONTRIBUTING.md)（如未创建，请先提 Issue 讨论）

## 免责声明

本项目仅提供技术研究和互操作实现。使用者必须：

- 仅连接本人所有或已获得明确授权的 Cursor 账号、订阅和运行环境
- 自行确认使用方式符合所在地法律法规、Cursor 服务条款、可接受使用政策
- 不得利用本项目共享或转售账号访问、规避额度或速率限制

详见 [DISCLAIMER.md](DISCLAIMER.md)
