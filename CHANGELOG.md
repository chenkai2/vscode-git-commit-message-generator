# Changelog

## [0.1.5] -2025-06-20

### 优化
- 降低最低VS Code版本要求至1.57

## [0.1.4] - 2025-06-05

### 新功能
- 支持多LLM提供商配置
  * 重构LLM配置，支持阿里云、OpenAI等9种提供商
  * 新增各提供商的URL、模型和API密钥配置项
  * 保留旧配置的向后兼容性
- 优化已删除文件的提示信息
- 添加各提供商的默认预设配置

### 优化
- 改进配置迁移逻辑
- 优化默认模型选择逻辑，提升用户体验
- 完善错误处理和提示信息

### 修复
- 修正已删除文件的不能正确生成提交信息的问题

## [0.1.3] - 2025-05-27

### 新功能
- 优化对删除文件的处理逻辑，添加文件历史内容和最后提交信息的获取
- 改进删除文件的提交信息生成，提供更丰富的上下文
- 添加对文件内容的智能截断，避免过大的内容影响性能

### 优化
- 优化状态栏提示信息，增加文件处理进度显示
- 改进错误处理机制，提供更友好的错误提示
- 优化本地备用提交信息生成逻辑

## [0.1.1] - 2025-03-25

### 新功能
- 添加对阿里云百炼API的支持，提供多种模型选择
  - deepseek-v3
  - deepseek-r1
  - qwen2.5-32b-instruct
  - deepseek-r1-distill-qwen-32b
  - qwen-plus
  - deepseek-r1-distill-llama-70b
  - qwen2-7b-instruct
- 添加对火山引擎API的支持，包含以下模型
  - deepseek-r1-250120
  - deepseek-r1-distill-qwen-32b-250120
  - deepseek-v3-241226
  - doubao-1-5-pro-256k-250115
- 优化本地部署Ollama的支持
- 完善README文档，添加详细的配置说明

### 修复
- 修复删除文件时的提交信息生成问题
- 优化错误处理和提示信息

## [0.1.0] - 2025-03-21

### 优化
- 优化代码清理逻辑，合并replace方法，简化正则表达式，提升代码可读性和维护性
- 优化LLM API调用逻辑，删除对anthropic和openrouter的支持，简化serviceName设置

### 新功能
- 添加LLM生成结果的最大token数量配置(1-8192，默认2048)
- 支持通过Git扩展获取仓库信息
- 增加状态栏提示信息和思考过程显示
- 启用流式处理以优化LLM API响应，提升用户体验
- 添加对多种LLM服务的支持：
  - siliconflow API
  - deepseek API
  - 腾讯云LLM

### 修复
- 修正API请求头授权处理逻辑
- 更新LLM API默认URL并添加协议配置

### 重构
- 重构LLM服务配置，统一协议字段和认证键名
- 优化请求数据处理逻辑，增加对多种协议的支持

## [0.0.1] - 2025-03-15

### 初始化
- 添加生成Git提交信息的扩展功能
- 添加TypeScript配置和Webpack配置文件
- 添加.vscodeignore和更新.gitignore文件
- 添加亮色和暗色模式SVG图标