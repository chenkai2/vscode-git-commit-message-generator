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
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate commit message based on staged changes",
	Long: `Generate a standardized commit message using AI models based on the staged changes in your Git repository.

This command analyzes the files in the staging area and their diffs, then uses an LLM service
to generate an appropriate commit message following conventional commit format.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取当前目录
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
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
				fmt.Fprintf(os.Stderr, "打开Git仓库时发生错误：%v\n", err)
				os.Exit(1)
			}

			// 获取父目录
			parent := filepath.Dir(gitRoot)
			if parent == gitRoot {
				fmt.Fprintf(os.Stderr, "错误：未找到Git仓库。\n请确保你在Git仓库目录或其子目录下执行此命令，或者使用 'git init' 初始化一个新的仓库。\n")
				os.Exit(1)
			}
			gitRoot = parent
		}

		// 更新工作目录为Git根目录
		cwd = gitRoot

		// 获取暂存的文件
		stagingFiles, diffContent, err := getStagedChanges(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取暂存的变更时发生错误：%v\n", err)
			os.Exit(1)
		}

		if len(stagingFiles) == 0 {
			fmt.Println("未找到暂存的文件。请先将文件添加到暂存区（git add）。")
			os.Exit(0)
		}

		fmt.Printf("找到 %d 个暂存的文件。\n\n", len(stagingFiles))

		// 生成commit message
		commitMsg, err := generateCommitMessage(stagingFiles, diffContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating commit message: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n\n------------------------")

		// 输出生成的commit message
		fmt.Println("\n暂存的文件:")
		for _, file := range stagingFiles {
			fmt.Println("- " + file)
		}
		fmt.Println("\n\n生成的提交信息:")
		fmt.Println("------------------------")
		fmt.Println(commitMsg)
		fmt.Println("------------------------")

		// 询问是否使用生成的commit message
		fmt.Print("您想用此消息提交吗? (默认为Y): [ Y / N]: ")
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
				fmt.Fprintf(os.Stderr, "Error executing git commit: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("提交成功！")
		} else {
			fmt.Println("取消提交。")
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// 获取Git配置
	getConfig := func(key string) string {
		cmd := exec.Command("git", "config", "--global", "--get", key)
		output, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(output))
	}

	// 设置Git配置项的默认值
	defaultConfig := map[string]map[string]interface{}{
		"commit-message-generator.llm": {
			"url":         "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
			"model":       "deepseek-v3",
			"prompt":      "根据以下Git变更生成Git提交信息，格式为 <type>: <description>。\n文件：${files}\n变更内容：${diff}",
			"system":      "标题行格式为 <type>: <description>，字数不要超过50个。两个换行后，输出正文内容，每个要点作为一个符号列表（Markdown格式），不超过70个字。标题和正文内容都用中文回答，不过type字段必须用英文。",
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
					fmt.Fprintf(os.Stderr, "Warning: Failed to write default config %s: %v\n", configKey, err)
				}
			}
			// 将Git配置同步到viper
			viper.Set(configKey, getConfig(configKey))
		}
	}

	// 添加命令行标志
	generateCmd.Flags().String("url", getConfig("commit-message-generator.llm.url"), "LLM API URL")
	generateCmd.Flags().String("model", getConfig("commit-message-generator.llm.model"), "LLM model name")
	generateCmd.Flags().String("prompt", getConfig("commit-message-generator.llm.prompt"), "Prompt template for commit message generation")
	generateCmd.Flags().String("system", getConfig("commit-message-generator.llm.system"), "System instruction for LLM")
	generateCmd.Flags().Float64("temperature", func() float64 {
		v, _ := strconv.ParseFloat(getConfig("commit-message-generator.llm.temperature"), 64)
		return v
	}(), "Temperature parameter for LLM (0-1)")
	generateCmd.Flags().Float64("top-p", func() float64 {
		v, _ := strconv.ParseFloat(getConfig("commit-message-generator.llm.top-p"), 64)
		return v
	}(), "Top-p parameter for LLM (0-1)")
	generateCmd.Flags().String("protocol", getConfig("commit-message-generator.llm.protocol"), "LLM API protocol (ollama or openai)")
	generateCmd.Flags().String("api-key", "", "API key for LLM service")
	generateCmd.Flags().Int("max-tokens", func() int { v, _ := strconv.Atoi(getConfig("commit-message-generator.llm.max-tokens")); return v }(), "Maximum tokens for LLM response")

	// 绑定标志到viper
	viper.BindPFlag("commit-message-generator.llm.url", generateCmd.Flags().Lookup("url"))
	viper.BindPFlag("commit-message-generator.llm.model", generateCmd.Flags().Lookup("model"))
	viper.BindPFlag("commit-message-generator.llm.prompt", generateCmd.Flags().Lookup("prompt"))
	viper.BindPFlag("commit-message-generator.llm.system", generateCmd.Flags().Lookup("system"))
	viper.BindPFlag("commit-message-generator.llm.temperature", generateCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("commit-message-generator.llm.top-p", generateCmd.Flags().Lookup("top-p"))
	viper.BindPFlag("commit-message-generator.llm.protocol", generateCmd.Flags().Lookup("protocol"))
	viper.BindPFlag("commit-message-generator.llm.api-key", generateCmd.Flags().Lookup("api-key"))
	viper.BindPFlag("commit-message-generator.llm.max-tokens", generateCmd.Flags().Lookup("max-tokens"))
}

// 获取暂存的文件和变更内容
func getStagedChanges(repoPath string) ([]string, string, error) {
	// 使用git命令获取暂存的文件列表
	cmd := exec.Command("git", "diff", "--name-only", "--cached")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get staged files: %v", err)
	}

	// 解析暂存的文件列表
	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, "", nil
	}

	// 获取每个文件的diff
	var allDiffs strings.Builder
	for _, file := range files {
		diffCmd := exec.Command("git", "diff", "--cached", file)
		diffCmd.Dir = repoPath

		diffOutput, err := diffCmd.Output()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get diff for file %s: %v", file, err)
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
		return "", fmt.Errorf("LLM API URL is not configured. Please set it in .gitconfig or provide --url flag")
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
		return "", fmt.Errorf("failed to parse API URL: %v", err)
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
		return "", fmt.Errorf("no matching LLM service found for URL %s and protocol %s", apiURL, protocol)
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
		return "", fmt.Errorf("unsupported protocol: %s", serviceConfig.Protocol)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
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
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("API认证失败")
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API请求失败，状态码 %d", resp.StatusCode)
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
			return "", fmt.Errorf("读取响应时发生错误：%v", err)
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
		return "", fmt.Errorf("no commit message generated")
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
