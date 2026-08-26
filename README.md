# Agent Skill Manager

一个跨平台的 **AI Agent 技能管理工具**。集中管理你的技能（Skills），并一键同步到 CodeBuddy、Codex 等 AI 编程工具。

## 功能特性

- **技能安装**：从 GitHub 仓库扫描并安装技能，也支持导入本地技能目录/文件
- **技能管理**：查看已安装技能、卸载、检查更新、一键更新
- **多 Agent 同步**：自动检测本机安装的 AI 工具（CodeBuddy、Codex 等），将技能同步到各工具的技能目录
- **共享技能库**：所有技能集中存放在一个共享目录中，统一管理
- **GitHub 集成**：支持 GitHub Token 认证，突破 API 限流

## 支持的 Agent

| Agent | 技能目录 |
|-------|---------|
| CodeBuddy | `~/.codebuddy/skills` |
| Codex | `~/.agents/skills` |

## 使用说明

1. 首次启动时，在「设置」页面选择共享技能库目录
2. 在「安装 Skill」页面从 GitHub 仓库或本地导入技能
3. 在「Agent 管理」页面检测并同步技能到已安装的 AI 工具

## 开发环境

- [Go](https://go.dev/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails](https://wails.io/) v2

## 编译

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式（热重载）
wails dev

# 编译生产版本
wails build
```

> 注意：macOS 版本需要在 macOS 上编译（需要 Xcode 工具链）。

## 许可证

[MIT](LICENSE)