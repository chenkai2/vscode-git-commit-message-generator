package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

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
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
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
					fmt.Fprintf(os.Stderr, "Warning: Failed to write default config %s: %v\n", configKey, err)
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
			helpMsg = "Ollama服务不需要API密钥，请确保Ollama服务已启动并可访问。"
		} else {
			apiURL := getConfig("commit-message-generator.llm.url")
			parsedURL, err := url.Parse(apiURL)
			if err == nil && parsedURL.Host != "" {
				helpMsg = fmt.Sprintf("请输入(%s) API 密钥：", parsedURL.Host)
			} else {
				helpMsg = "请输入API密钥："
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
					fmt.Fprintf(os.Stderr, "Error saving API key: %v\n", err)
				} else {
					// 更新Viper的内存配置
					viper.Set("commit-message-generator.llm.api-key", apiKey)
					fmt.Println("API密钥已成功保存")
				}
			}
		} else {
			helpMsg += "\n或者切换到其他LLM服务：git config --global commit-message-generator.llm.url 'new-api-url'"
			fmt.Fprintln(os.Stderr, helpMsg)
		}
	}
	viper.AutomaticEnv() // read in environment variables that match
}
