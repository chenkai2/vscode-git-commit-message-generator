# Git Commit Message Generator

<p align="center">
  <img src="media/panda-avatar.png" alt="Git Commit Message Generator Logo" width="128" height="128">
</p>

一个强大的Git提交信息生成器，基于AI模型自动分析暂存的代码变更并生成规范的commit message。

## 功能特点

- 🤖 基于AI模型自动分析代码变更
- 🔄 支持多种LLM服务（Ollama、OpenAI、阿里云百炼、火山引擎等）
- 🌍 支持中英文等多语言提交信息
- ⚙️ 可自定义提示词模板和参数配置
- 🎨 优雅的用户界面和交互体验
- 🚀 展示推理过程，支持本地部署的Ollama

## 安装

1. 在VSCode中打开扩展市场
2. 搜索"Git Commit Message Generator"
3. 点击安装即可

## 使用方法

1. 在设置中配置AI服务的API相关信息
   - 默认使用阿里云百炼的AI接口，模型为`deepseek-v3`
     - 获取API密钥：[阿里云百炼](https://bailian.console.aliyun.com/?apiKey=1#/api-key)
     - 生成密钥后，可以直接使用各种模型，新用户半年内每种模型免费 100w tokens，可以用的模型有：
       - `deepseek-v3`
       - `deepseek-r1`
       - `qwen2.5-32b-instruct`
       - `deepseek-r1-distill-qwen-32b`
       - `qwen-plus`
       - `deepseek-r1-distill-llama-70b` 这个模型 free，只是用的人太多有点慢
       - `qwen2-7b-instruct`
   - 其次推荐[火山引擎](https://console.volcengine.com/ark/region:ark+cn-beijing/apiKey?apikey=%7B%7D)，截止2025年5月31日，每天每个模型免费 50w tokens
     - 生成api后需要手动开通需要开通的模型
     - 支持的模型较少，只有deepseek系的和doubao系的，比如：
     - `deepseek-r1-250120` 每天50w tokens
     - `deepseek-r1-distill-qwen-32b-250120` 每天50w tokens
     - `deepseek-v3-241226` 一共100w tokens
     - `doubao-1-5-pro-256k-250115` 每天50w tokens
   - 支持其他兼容openai接口的大模型服务，比如腾讯元宝、Anthropic、硅基流动、DeepSeek等
   - 本扩展支持本地部署的Ollama，只需要把 protocol 改为 ollama，url 改为`http://localhost:11434/api/generate`即可
2. 在Git源代码管理视图中，将要提交的文件添加到暂存区
3. 点击工具栏中的"生成 Commit Message"按钮
4. 插件会自动分析暂存的代码变更，并生成规范的提交信息
5. DeepSeek等有推理过程的大模型，会在状态栏显示推理过程

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

- Ollama（本地部署）
- OpenAI
- 阿里云百炼
- 火山引擎
- Anthropic
- 腾讯混元
- DeepSeek
- SiliconFlow
- 其他兼容OpenAI接口的服务

## 贡献

欢迎提交问题和功能请求！如果您想贡献代码，请随时提交PR。

## 许可证

MIT License