# Changelog

## [0.1.0] - 2024-03-21

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

## [0.0.1] - 2024-03-15

### 初始化
- 添加生成Git提交信息的扩展功能
- 添加TypeScript配置和Webpack配置文件
- 添加.vscodeignore和更新.gitignore文件
- 添加亮色和暗色模式SVG图标