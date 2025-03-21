# Git Commit Message Generator

一个强大的Git提交信息生成器，基于AI模型自动分析暂存的代码变更并生成规范的commit message。

## 功能特点

- 🤖 基于AI模型自动分析代码变更
- 🔄 支持多种LLM服务（Ollama、OpenAI等）
- ⚙️ 可自定义提示词模板和参数配置
- 🎨 优雅的用户界面和交互体验

## 安装

1. 在VSCode中打开扩展市场
2. 搜索"Git Commit Message Generator"
3. 点击安装即可

## 使用方法

1. 在Git源代码管理视图中，将要提交的文件添加到暂存区
2. 点击工具栏中的"生成Commit Message"按钮
3. 插件会自动分析暂存的代码变更，并生成规范的提交信息

## 配置选项

在VSCode设置中，可以自定义以下配置：

- `llm.url`: LLM API的URL地址
- `llm.model`: 使用的AI模型
- `llm.prompt`: 生成提交信息的提示词模板
- `llm.system`: 系统指令
- `llm.temperature`: 生成结果的随机性（0-1）
- `llm.top_p`: 采样时的累积概率阈值（0-1）
- `llm.apiKey`: API密钥
- `llm.protocol`: API协议（ollama/openai）
- `llm.max_tokens`: 生成结果的最大token数量

## 支持的LLM服务

- Ollama
- OpenAI
- 阿里云百炼
- Anthropic
- 腾讯混元
- DeepSeek
- SiliconFlow

## 许可证

MIT License