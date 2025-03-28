package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chenkai2/git-commitx/i18n"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "git-commitx",
	Short: "AI-powered Git commit message generator",
	Long: `Git Commitx is a powerful Git commit message generator that uses AI models
to automatically analyze staged code changes and generate standardized commit messages.

It supports various LLM services like Ollama, OpenAI, and more, with customizable
prompt templates and configuration options.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取当前目录
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("error.get_current_dir", err)+"\n")
			os.Exit(1)
		}

		// 向上递归查找Git仓库
		gitRoot := cwd
		for {
			_, err = git.PlainOpen(gitRoot)
			if err == nil {
				break
			}
			if err != git.ErrRepositoryNotExists {
				fmt.Fprintf(os.Stderr, i18n.T("error.open_git_repo", err)+"\n")
				os.Exit(1)
			}

			// 获取父目录
			parent := filepath.Dir(gitRoot)
			if parent == gitRoot {
				fmt.Fprintf(os.Stderr, i18n.T("error.no_git_repo")+"\n")
				os.Exit(1)
			}
			gitRoot = parent
		}

		// 更新工作目录为Git根目录
		cwd = gitRoot

		// 获取暂存的文件
		stagingFiles, diffContent, err := getStagedChanges(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("error.get_staged_changes", err)+"\n")
			os.Exit(1)
		}

		if len(stagingFiles) == 0 {
			fmt.Println(i18n.T("info.no_staged_files"))
			os.Exit(0)
		}

		fmt.Printf(i18n.T("info.found_staged_files", len(stagingFiles)) + "\n\n")

		// 生成commit message
		commitMsg, err := generateCommitMessage(stagingFiles, diffContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("error.generate_commit_msg", err)+"\n")
			os.Exit(1)
		}

		fmt.Println("\n\n------------------------")

		// 输出生成的commit message
		fmt.Println("\n" + i18n.T("info.staged_files"))
		for _, file := range stagingFiles {
			fmt.Println("- " + file)
		}
		fmt.Println("\n\n" + i18n.T("info.generated_commit_msg"))
		fmt.Println("------------------------")
		fmt.Println(commitMsg)
		fmt.Println("------------------------")

		// 询问是否使用生成的commit message
		fmt.Print(i18n.T("prompt.use_commit_msg"))
		var response string
		fmt.Scanln(&response)

		if response == "" || strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			// 使用生成的commit message进行提交
			cmd := exec.Command("git", "commit", "-m", commitMsg)
			cmd.Dir = cwd
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, i18n.T("error.execute_git_commit", err)+"\n")
				os.Exit(1)
			}

			fmt.Println(i18n.T("info.commit_success"))
		} else {
			fmt.Println(i18n.T("info.commit_cancelled"))
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

// LLM服务配置
type LLMService struct {
	Name     string
	Protocol string
	Hostname string
	APIPath  string
	Headers  map[string]string
	AuthKey  string
}

// LLM API请求结构
type OpenAIRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	TopP        float64       `json:"top-p"`
	MaxTokens   int           `json:"max-tokens"`
	Stream      bool          `json:"stream"`
}

type OllamaRequest struct {
	Model       string  `json:"model"`
	System      string  `json:"system"`
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top-p"`
	MaxTokens   int     `json:"max-tokens"`
	Stream      bool    `json:"stream"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI响应结构
type OpenAIResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// Ollama响应结构
type OllamaResponse struct {
	Response string `json:"response"`
}

// 获取暂存的文件和变更内容
func getStagedChanges(repoPath string) ([]string, string, error) {
	// 使用git命令获取暂存的文件列表
	cmd := exec.Command("git", "diff", "--name-only", "--cached")
	cmd.Dir = repoPath

	// 设置标准错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf(i18n.T("error.get_staged_files", err, stderr.String()))
	}

	// 解析暂存的文件列表
	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, "", nil
	}

	// 获取每个文件的diff
	var allDiffs strings.Builder
	for _, file := range files {
		diffCmd := exec.Command("git", "diff", "--cached", "--", file)
		diffCmd.Dir = repoPath

		// 设置标准错误输出
		var diffStderr bytes.Buffer
		diffCmd.Stderr = &diffStderr

		diffOutput, err := diffCmd.Output()
		if err != nil {
			return nil, "", fmt.Errorf(i18n.T("error.get_file_diff", file, err, diffStderr.String()))
		}

		allDiffs.WriteString(fmt.Sprintf("\n文件: %s\n%s\n", file, string(diffOutput)))
	}

	return files, allDiffs.String(), nil
}

// 生成commit message
func generateCommitMessage(stagedFiles []string, diffContent string) (string, error) {
	// 获取配置
	apiURL := viper.GetString("commit-message-generator.llm.url")
	if apiURL == "" {
		return "", fmt.Errorf(i18n.T("error.llm_api_url_not_configured"))
	}

	model := viper.GetString("commit-message-generator.llm.model")
	temperature := viper.GetFloat64("commit-message-generator.llm.temperature")
	topP := viper.GetFloat64("commit-message-generator.llm.top-p")
	protocol := viper.GetString("commit-message-generator.llm.protocol")
	maxTokens := viper.GetInt("commit-message-generator.llm.max-tokens")
	apiKey := viper.GetString("commit-message-generator.llm.api-key")

	// 获取提示词模板和系统指令
	promptTemplate := viper.GetString("commit-message-generator.llm.prompt")
	system := viper.GetString("commit-message-generator.llm.system")

	// 替换模板变量
	prompt := strings.ReplaceAll(promptTemplate, "${files}", strings.Join(stagedFiles, "\n"))
	prompt = strings.ReplaceAll(prompt, "${diff}", diffContent)

	// 定义LLM服务配置
	llmServices := []LLMService{
		{
			Name:     "ollama",
			Protocol: "ollama",
			Hostname: "localhost",
			APIPath:  "/api/generate",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:     "openai",
			Protocol: "openai",
			Hostname: "api.openai.com",
			APIPath:  "/chat/completions",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer ",
			},
			AuthKey: "Authorization",
		},
		{
			Name:     "aliyun",
			Protocol: "openai",
			Hostname: "dashscope.aliyuncs.com",
			APIPath:  "/chat/completions",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer ",
			},
			AuthKey: "Authorization",
		},
		{
			Name:     "anthropic",
			Protocol: "openai",
			Hostname: "api.anthropic.com",
			APIPath:  "/chat/completions",
			Headers: map[string]string{
				"Content-Type":      "application/json",
				"Authorization":     "Bearer ",
				"anthropic-version": "2023-06-01",
			},
			AuthKey: "x-api-key",
		},
		{
			Name:     "tencent",
			Protocol: "openai",
			Hostname: "api.hunyuan.cloud.tencent.com",
			APIPath:  "/chat/completions",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer ",
			},
			AuthKey: "Authorization",
		},
		{
			Name:     "deepseek",
			Protocol: "openai",
			Hostname: "api.deepseek.com",
			APIPath:  "/chat/completions",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer ",
			},
			AuthKey: "Authorization",
		},
		{
			Name:     "siliconflow",
			Protocol: "openai",
			Hostname: "api.siliconflow.cn",
			APIPath:  "/chat/completions",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer ",
			},
			AuthKey: "Authorization",
		},
	}

	// 解析URL
	parsedURL, err := parseURL(apiURL)
	if err != nil {
		return "", fmt.Errorf(i18n.T("error.parse_api_url", err))
	}

	// 获取匹配的服务配置
	var serviceConfig *LLMService
	for _, service := range llmServices {
		if service.Hostname == parsedURL.Hostname() {
			serviceConfig = &service
			break
		}
	}

	if serviceConfig == nil {
		// 如果没有匹配的服务，根据协议选择默认服务
		for _, service := range llmServices {
			if service.Protocol == protocol {
				serviceConfig = &service
				break
			}
		}
	}

	if serviceConfig == nil {
		return "", fmt.Errorf(i18n.T("error.no_matching_llm_service", apiURL, protocol))
	}

	// 准备请求数据
	var requestBody []byte
	switch serviceConfig.Protocol {
	case "ollama":
		request := OllamaRequest{
			Model:       model,
			System:      system,
			Prompt:      prompt,
			Temperature: temperature,
			TopP:        topP,
			MaxTokens:   maxTokens,
			Stream:      true,
		}
		requestBody, err = json.Marshal(request)
	case "openai", "anthropic":
		request := OpenAIRequest{
			Model: model,
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: system,
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
			Temperature: temperature,
			TopP:        topP,
			MaxTokens:   maxTokens,
			Stream:      true,
		}
		requestBody, err = json.Marshal(request)
	default:
		return "", fmt.Errorf(i18n.T("error.unsupported_protocol", serviceConfig.Protocol))
	}

	if err != nil {
		return "", fmt.Errorf(i18n.T("error.marshal_request", err))
	}

	// 创建请求选项
	req := &http.Request{
		Method: "POST",
		URL:    parsedURL,
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewBuffer(requestBody)),
	}

	// 设置请求头
	for key, value := range serviceConfig.Headers {
		req.Header.Set(key, value)
	}

	// 添加API密钥
	if apiKey != "" && serviceConfig.AuthKey != "" {
		// 对于需要Bearer auth的服务，确保正确添加Bearer前缀
		if serviceConfig.Protocol == "openai" || serviceConfig.Protocol == "anthropic" {
			req.Header.Set(serviceConfig.AuthKey, "Bearer "+apiKey)
		} else {
			req.Header.Set(serviceConfig.AuthKey, apiKey)
		}
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf(i18n.T("error.send_http_request", err))
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf(i18n.T("error.api_auth_failed"))
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(i18n.T("error.api_request_failed", resp.StatusCode))
	}

	// 用于存储完整的提交信息
	var commitMessage strings.Builder
	var reasoningContent strings.Builder

	// 处理流式响应
	for {
		// 读取一行数据
		buf := make([]byte, 4096)
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf(i18n.T("error.read_response", err))
		}
		if n == 0 {
			break
		}

		// 将数据按行分割
		lines := strings.Split(string(buf[:n]), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "data: [DONE]" {
				continue
			}

			// 如果是SSE格式，去掉"data: "前缀
			line = strings.TrimPrefix(line, "data: ")

			// 尝试解析JSON
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(line), &response); err != nil {
				// 如果是语法错误或其他JSON解析错误，记录详细信息并继续
				// fmt.Fprintf(os.Stderr, "警告：跳过无效的JSON数据行：%v，原始数据：%s\n", err, line)
				continue
			}

			// 检查响应格式是否有效
			if len(response) == 0 {
				continue
			}

			switch serviceConfig.Protocol {
			case "ollama":
				if responseText, ok := response["response"].(string); ok && responseText != "" {
					commitMessage.WriteString(responseText)
					fmt.Print(responseText)
				}
			case "openai":
				if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						// 处理不同的响应格式
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							// 处理推理内容
							if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
								reasoningContent.WriteString(reasoning)
								fmt.Print(reasoning)
							}
							// 处理生成的文本
							if content, ok := delta["content"].(string); ok && content != "" {
								commitMessage.WriteString(content)
								fmt.Print(content)
							}
						} else if message, ok := choice["message"].(map[string]interface{}); ok {
							// 处理非流式响应格式
							if content, ok := message["content"].(string); ok && content != "" {
								commitMessage.WriteString(content)
								fmt.Print(content)
							}
						}
					}
				}
			}
		}
	}

	if commitMessage.Len() == 0 {
		return "", fmt.Errorf(i18n.T("error.no_commit_msg"))
	}

	return strings.Trim(commitMessage.String(), "\n"), nil
}

// 解析URL
func parseURL(urlStr string) (*url.URL, error) {
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, err
	}
	return parsedURL, nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// 添加命令行标志
	rootCmd.Flags().String("url", "", "LLM API URL")
	rootCmd.Flags().String("model", "", "LLM model name")
	rootCmd.Flags().String("prompt", "", "Prompt template for commit message generation")
	rootCmd.Flags().String("system", "", "System instruction for LLM")
	rootCmd.Flags().Float64("temperature", 0.7, "Temperature parameter for LLM (0-1)")
	rootCmd.Flags().Float64("top-p", 0.9, "Top-p parameter for LLM (0-1)")
	rootCmd.Flags().String("protocol", "", "LLM API protocol (ollama or openai)")
	rootCmd.Flags().String("api-key", "", "API key for LLM service")
	rootCmd.Flags().Int("max-tokens", 2048, "Maximum tokens for LLM response")

	// 绑定标志到viper
	viper.BindPFlag("commit-message-generator.llm.url", rootCmd.Flags().Lookup("url"))
	viper.BindPFlag("commit-message-generator.llm.model", rootCmd.Flags().Lookup("model"))
	viper.BindPFlag("commit-message-generator.llm.prompt", rootCmd.Flags().Lookup("prompt"))
	viper.BindPFlag("commit-message-generator.llm.system", rootCmd.Flags().Lookup("system"))
	viper.BindPFlag("commit-message-generator.llm.temperature", rootCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("commit-message-generator.llm.top-p", rootCmd.Flags().Lookup("top-p"))
	viper.BindPFlag("commit-message-generator.llm.protocol", rootCmd.Flags().Lookup("protocol"))
	viper.BindPFlag("commit-message-generator.llm.api-key", rootCmd.Flags().Lookup("api-key"))
	viper.BindPFlag("commit-message-generator.llm.max-tokens", rootCmd.Flags().Lookup("max-tokens"))
}

// initConfig reads in Git config and ENV variables if set.
func initConfig() {
	// 获取Git配置
	getConfig := func(key string) string {
		cmd := exec.Command("git", "config", "--global", "--get", key)
		output, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(output))
	}

	// 定义默认配置
	defaultConfig := map[string]map[string]interface{}{
		"commit-message-generator.llm": {
			"url":         "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
			"model":       "deepseek-v3",
			"prompt":      "根据以下Git变更生成Git提交信息，格式为 <type>: <description>。\n文件：${files}\n变更内容：${diff}",
			"system":      "标题行格式为 <type>: <description>，字数不要超过50个，description如果不是中文，则翻译成中文。两个换行后，输出正文内容，每个要点作为一个符号列表，不超过70个字。type使用英文，description和正文用中文，如果不是，则翻译成中文。要点简洁清晰。",
			"temperature": 0.7,
			"top-p":       0.9,
			"protocol":    "openai",
			"max-tokens":  2048,
			"api-key":     "",
		},
	}

	// 检查并设置默认配置
	for section, values := range defaultConfig {
		for key, value := range values {
			configKey := fmt.Sprintf("%s.%s", section, key)
			if getConfig(configKey) == "" {
				// 如果配置不存在，写入默认值
				cmd := exec.Command("git", "config", "--global", configKey, fmt.Sprintf("%v", value))
				if err := cmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, i18n.T("error.write_default_config", configKey, err)+"\n")
				}
			}
			// 将Git配置同步到viper
			viper.Set(configKey, getConfig(configKey))
		}
	}
	if getConfig("commit-message-generator.llm.api-key") == "" {
		// 根据服务类型提供友好的API密钥设置提示
		var helpMsg string
		protocol := getConfig("commit-message-generator.llm.protocol")
		if protocol == "ollama" {
			helpMsg = i18n.T("prompt.ollama_no_key")
		} else {
			apiURL := getConfig("commit-message-generator.llm.url")
			parsedURL, err := url.Parse(apiURL)
			if err == nil && parsedURL.Host != "" {
				helpMsg = i18n.T("prompt.enter_api_key_for", parsedURL.Host)
			} else {
				helpMsg = i18n.T("prompt.enter_api_key")
			}
		}

		if helpMsg != "" && !strings.Contains(helpMsg, "Ollama") {
			fmt.Print(helpMsg)
			var apiKey string
			fmt.Scanln(&apiKey)

			if apiKey != "" {
				// 将API密钥保存到Git配置
				cmd := exec.Command("git", "config", "--global", "commit-message-generator.llm.api-key", apiKey)
				if err := cmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, i18n.T("error.save_api_key", err)+"\n")
				} else {
					// 更新Viper的内存配置
					viper.Set("commit-message-generator.llm.api-key", apiKey)
					fmt.Println(i18n.T("info.api_key_saved"))
				}
			}
		} else {
			helpMsg += "\n" + i18n.T("prompt.switch_llm_service")
			fmt.Fprintln(os.Stderr, helpMsg)
		}
	}
	viper.AutomaticEnv() // read in environment variables that match
}
