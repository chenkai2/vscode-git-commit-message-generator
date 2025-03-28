package i18n

type Message struct {
	Key    string
	ZhHans string
	EnUS   string
}

var Messages = []Message{
	{
		Key:    "error.get_current_dir",
		ZhHans: "获取当前目录时发生错误：%v",
		EnUS:   "Error getting current directory: %v",
	},
	{
		Key:    "error.open_git_repo",
		ZhHans: "打开Git仓库时发生错误：%v",
		EnUS:   "Error opening Git repository: %v",
	},
	{
		Key:    "error.no_git_repo",
		ZhHans: "错误：未找到Git仓库。\n请确保你在Git仓库目录或其子目录下执行此命令，或者使用 'git init' 初始化一个新的仓库。",
		EnUS:   "Error: Git repository not found.\nPlease ensure you are in a Git repository directory or its subdirectory, or use 'git init' to initialize a new repository.",
	},
	{
		Key:    "error.get_staged_changes",
		ZhHans: "获取暂存的变更时发生错误：%v",
		EnUS:   "Error getting staged changes: %v",
	},
	{
		Key:    "info.no_staged_files",
		ZhHans: "未找到暂存的文件。请先将文件添加到暂存区（git add）。",
		EnUS:   "No staged files found. Please add files to staging area first (git add).",
	},
	{
		Key:    "info.found_staged_files",
		ZhHans: "找到 %d 个暂存的文件。",
		EnUS:   "Found %d staged files.",
	},
	{
		Key:    "info.staged_files",
		ZhHans: "暂存的文件:",
		EnUS:   "Staged files:",
	},
	{
		Key:    "info.generated_commit_msg",
		ZhHans: "生成的提交信息:",
		EnUS:   "Generated commit message:",
	},
	{
		Key:    "prompt.use_commit_msg",
		ZhHans: "您想用此消息提交吗? (默认为Y): [ Y / N]: ",
		EnUS:   "Do you want to commit with this message? (default is Y): [ Y / N]: ",
	},
	{
		Key:    "info.commit_success",
		ZhHans: "提交成功！",
		EnUS:   "Commit successful!",
	},
	{
		Key:    "info.commit_cancelled",
		ZhHans: "取消提交。",
		EnUS:   "Commit cancelled.",
	},
	{
		Key:    "error.api_auth_failed",
		ZhHans: "API认证失败",
		EnUS:   "API authentication failed",
	},
	{
		Key:    "error.api_request_failed",
		ZhHans: "API请求失败，状态码 %d",
		EnUS:   "API request failed with status code %d",
	},
	{
		Key:    "error.read_response",
		ZhHans: "读取响应时发生错误：%v",
		EnUS:   "Error reading response: %v",
	},
	{
		Key:    "error.no_commit_msg",
		ZhHans: "未生成提交信息",
		EnUS:   "No commit message generated",
	},
	{
		Key:    "error.generate_commit_msg",
		ZhHans: "生成提交信息时发生错误：%v",
		EnUS:   "Error generating commit message: %v",
	},
	{
		Key:    "error.execute_git_commit",
		ZhHans: "执行git commit时发生错误：%v",
		EnUS:   "Error executing git commit: %v",
	},
	{
		Key:    "error.write_default_config",
		ZhHans: "写入默认配置 %s 时发生错误：%v",
		EnUS:   "Warning: Failed to write default config %s: %v",
	},
	{
		Key:    "error.save_api_key",
		ZhHans: "保存API密钥时发生错误：%v",
		EnUS:   "Error saving API key: %v",
	},
	{
		Key:    "info.api_key_saved",
		ZhHans: "API密钥已成功保存",
		EnUS:   "API key has been successfully saved",
	},
	{
		Key:    "prompt.ollama_no_key",
		ZhHans: "Ollama服务不需要API密钥，请确保Ollama服务已启动并可访问。",
		EnUS:   "Ollama service does not require an API key, please ensure the Ollama service is running and accessible.",
	},
	{
		Key:    "prompt.enter_api_key_for",
		ZhHans: "请输入(%s) API 密钥：",
		EnUS:   "Please enter API key for (%s):",
	},
	{
		Key:    "prompt.enter_api_key",
		ZhHans: "请输入API密钥：",
		EnUS:   "Please enter API key:",
	},
	{
		Key:    "prompt.switch_llm_service",
		ZhHans: "或者切换到其他LLM服务：git config --global commit-message-generator.llm.url 'new-api-url'",
		EnUS:   "Or switch to another LLM service: git config --global commit-message-generator.llm.url 'new-api-url'",
	},
	{
		Key:    "error.get_staged_files",
		ZhHans: "获取暂存文件列表失败：%v\n%s",
		EnUS:   "Failed to get staged file list: %v\n%s",
	},
	{
		Key:    "error.get_file_diff",
		ZhHans: "获取文件 %s 的差异失败：%v\n%s",
		EnUS:   "Failed to get diff for file %s: %v\n%s",
	},
	{
		Key:    "error.llm_api_url_not_configured",
		ZhHans: "LLM API URL未配置。请在.gitconfig中设置或提供--url标志",
		EnUS:   "LLM API URL is not configured. Please set it in .gitconfig or provide --url flag",
	},
	{
		Key:    "error.parse_api_url",
		ZhHans: "解析API URL失败：%v",
		EnUS:   "Failed to parse API URL: %v",
	},
	{
		Key:    "error.no_matching_llm_service",
		ZhHans: "未找到匹配URL %s 和协议 %s 的LLM服务",
		EnUS:   "No matching LLM service found for URL %s and protocol %s",
	},
	{
		Key:    "error.unsupported_protocol",
		ZhHans: "不支持的协议：%s",
		EnUS:   "Unsupported protocol: %s",
	},
	{
		Key:    "error.marshal_request",
		ZhHans: "序列化请求失败：%v",
		EnUS:   "Failed to marshal request: %v",
	},
	{
		Key:    "error.send_http_request",
		ZhHans: "发送HTTP请求失败：%v",
		EnUS:   "Failed to send HTTP request: %v",
	},
}
