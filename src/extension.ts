import * as vscode from 'vscode';
import { simpleGit, SimpleGit } from 'simple-git';
import * as https from 'https';
import * as http from 'http';
import * as url from 'url';
import { log } from 'console';
import { match } from 'assert';

export function activate(context: vscode.ExtensionContext) {
  console.log('插件 "vscode-git-commit-message-generator" 已激活');

  // 注册命令
  let disposable = vscode.commands.registerCommand('vscode-git-commit-message-generator.generateCommitMessage', async (sourceControl) => {
    try {
      // 获取Git扩展
      const gitExtension = vscode.extensions.getExtension('vscode.git')?.exports;
      if (!gitExtension) {
        vscode.window.showErrorMessage('无法获取Git扩展');
        return;
      }

      const api = gitExtension.getAPI(1);
      if (!api) {
        vscode.window.showErrorMessage('无法获取Git API');
        return;
      }

      // 获取当前点击的Git源
      const repository = sourceControl?.rootUri
        ? api.repositories.find((repo: { rootUri: { fsPath: string } }) => repo.rootUri.fsPath === sourceControl.rootUri.fsPath)
        : api.repositories[0];

      if (!repository) {
        vscode.window.showErrorMessage('无法获取Git仓库');
        return;
      }

      const rootPath = repository.rootUri.fsPath;
      const git: SimpleGit = simpleGit(rootPath);

      // 检查是否有staged文件
      const status = await git.status();
      if (status.staged.length === 0) {
        vscode.window.showWarningMessage('没有暂存的文件，请先添加文件到暂存区');
        return;
      }

      // 创建状态栏消息
      const statusBarMessage = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left);
      // 获取暂存区的文件变更
      const stagedFiles = status.staged;
      statusBarMessage.text = `找到 ${stagedFiles.length} 个暂存的文件`;
      context.subscriptions.push(statusBarMessage);
      statusBarMessage.show();

      // 获取每个文件的diff
      let allDiffs = '';
      // 分类文件 - 使用status.deleted和status.staged的交集确定删除文件
      const deletedFiles = stagedFiles.filter(file => status.deleted.includes(file));
      // 其他文件是staged文件减去deletedFiles
      const otherFiles = stagedFiles.filter(file => !deletedFiles.includes(file));
      
      // 处理非删除文件
      for (const file of otherFiles) {
        try {
          const diff = await git.diff(['--cached', file]);
          allDiffs += `\n文件: ${file}\n${diff}\n`;
        } catch (error) {
          console.error(`获取文件 ${file} 的diff失败:`, error);
        }
      }
      
      // 特殊处理删除的文件
      for (const file of deletedFiles) {
        try {
          // 尝试获取删除文件的基本信息
          const fileName = file || '';
          // 尝试获取文件的最后一次提交信息，了解文件的用途
          let fileInfo = '';
          let fileContent = '';
          try {
            // 获取文件的最后一次提交日志
            const log = await git.log({ file: fileName, maxCount: 1 });
            if (log.all.length > 0) {
              const lastCommit = log.all[0];
              fileInfo = `\n该文件最后一次提交信息: ${lastCommit.message}\n`;
              
              // 尝试获取文件在最后一次提交前的内容
              try {
                // 使用git show命令获取文件的历史版本内容
                const fileHistoryContent = await git.raw(['show', `${lastCommit.hash}:${fileName}`]);
                if (fileHistoryContent) {
                  // 限制文件内容长度，避免过大
                  const maxContentLength = 300; // 最多300个字符
                  const lines = fileHistoryContent.split('\n');
                  const first15Lines = lines.slice(0, 15).join('\n'); // 最多取15行
                  
                  // 同时满足字符数和行数限制
                  let truncatedContent = first15Lines;
                  if (truncatedContent.length > maxContentLength) {
                    truncatedContent = truncatedContent.substring(0, maxContentLength) + '\n... (内容过长已截断)';
                  } else if (lines.length > 15) {
                    truncatedContent += '\n... (只显示前15行)';
                  }
                  
                  fileContent = `\n该文件删除的内容:\n\`\`\`\n${truncatedContent}\n\`\`\`\n`;
                }
              } catch (showError) {
                console.log(`获取文件 ${fileName} 的历史内容失败:`, showError);
              }
            }
          } catch (logError) {
            console.log(`获取文件 ${fileName} 的提交历史失败:`, logError);
          }
          
          // 添加删除文件的上下文信息
          allDiffs += `\n已删除的文件: ${file} ${fileInfo}${fileContent}\n`;
        } catch (error) {
          console.error(`获取删除文件 ${file} 的信息失败:`, error);
          // 即使获取失败，也添加基本信息
          allDiffs += `\n文件: ${file} (已删除，无法获取更多信息)\n`;
        }
      }

      statusBarMessage.text = 'AI正在生成commit message...';

      let commitMessage = '';
      try {
        // 根据变更内容生成commit message
        // 为已删除文件添加后缀
        const markedFiles = stagedFiles.map(file => 
          deletedFiles.includes(file) ? `${file}（已删除）` : file
        );
        commitMessage = await generateCommitMessage(markedFiles, allDiffs, repository.inputBox, statusBarMessage);
        statusBarMessage.dispose();
      } catch (error) {
        statusBarMessage.dispose();
        throw error;
      }

      // 设置最终的commit message
      repository.inputBox.value = commitMessage;
      statusBarMessage.text = '已设置commit message';
    } catch (error) {
      console.error('生成commit message时出错:', error);
      vscode.window.showErrorMessage(`生成commit message失败: ${error}`);
    }
  });

  context.subscriptions.push(disposable);
}

/**
 * 调用LLM API生成commit message
 */
async function callLLMAPI(stagedFiles: string[], diffContent: string, inputBox: any, statusBarMessage: vscode.StatusBarItem): Promise<string> {
  const modelServices = [
    {
      name: 'ollama',
      // @doc https://github.com/ollama/ollama/blob/main/docs/api.md#chat
      protocol: 'ollama',
      hostname: 'localhost',
      apiSuffix: '/api/generate',
      headers: {
        'Content-Type': 'application/json'
      }
    },
    {
      name: 'openai',
      // @doc https://platform.openai.com/docs/api-reference/completions/create
      hostname: 'api.openai.com',
      protocol: 'openai',
      apiSuffix: '/chat/completions',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer '
      },
      AuthKey: 'Authorization'
    },
    {
      name: 'aliyun',
      // @doc https://bailian.console.aliyun.com/#/model-market/detail/qwen2.5-32b-instruct?tabKey=sdk
      hostname: 'dashscope.aliyuncs.com',
      protocol: 'openai',
      apiSuffix: '/chat/completions',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer '
      },
      AuthKey: 'Authorization'
    },
    {
      name: 'anthropic',
      // @doc https://docs.anthropic.com/en/api/getting-started
      hostname: 'api.anthropic.com',
      protocol: 'anthropic',
      apiSuffix: '/chat/completions',
      headers: {
        'Content-Type': 'application/json',
        "Authorization": "Bearer ",
        'anthropic-version': '2023-06-01'
      },
      AuthKey: 'x-api-key'
    },
    {
      name: 'tencent',
      //@doc https://cloud.tencent.com/document/product/1729/111007
      hostname: 'api.hunyuan.cloud.tencent.com',
      protocol: 'openai',
      apiSuffix: '/chat/completions',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer '
      },
      AuthKey: 'Authorization'
    },
    {
      name: 'deepseek',
      hostname: 'api.deepseek.com',
      apiSuffix: '/chat/completions',
      protocol: 'openai',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer '
      },
      AuthKey: 'Authorization'
    },
    {
      name: 'siliconflow',
      hostname: 'api.siliconflow.cn',
      protocol: 'openai',
      apiSuffix: '/chat/completions',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer '
      },
      AuthKey: 'Authorization'
    }
  ];
  // 获取配置
  const config = vscode.workspace.getConfiguration('vscode-git-commit-message-generator');
  const provider = config.get<string>('llm.provider') || 'aliyun';
  var apiUrl = '';
  var model = '';
  var apiKey = '';
  var protocol = 'openai';
  const temperature = config.get<number>('llm.temperature') || 0.7;
  const topP = config.get<number>('llm.top_p') || 1;
  const maxTokens = config.get<number>('llm.max_tokens') || 2048;
  if (provider == "custom") {
    apiUrl = config.get<string>(`providers.${provider}.url`) || '';
    const oldApiUrl = config.get<string>('llm.url') || '';
    const oldProtocol = config.get<string>('llm.protocol') || '';
    const oldModel = config.get<string>('llm.model') || '';
    const oldApiKey = config.get<string>('llm.apiKey') || '';
    if (apiUrl == '' && oldApiUrl != '') {
      apiUrl = oldApiUrl;
      try {
        config.update(`providers.${provider}.url`, oldApiUrl, vscode.ConfigurationTarget.Global);
        config.update(`providers.${provider}.protocol`, oldProtocol, vscode.ConfigurationTarget.Global);
        config.update(`providers.${provider}.apiKey`, oldApiKey, vscode.ConfigurationTarget.Global);
        config.update(`providers.${provider}.model`, oldModel, vscode.ConfigurationTarget.Global);
      } catch (error) {
        console.error('更新配置失败:', error);
        vscode.window.showErrorMessage(`同步 ${provider} 配置失败: ${error}`);
      }
    }
    model = config.get<string>(`providers.${provider}.model`) || oldModel || '';
    protocol = config.get<string>(`providers.${provider}.protocol`) || oldProtocol || 'openai';
    apiKey = config.get<string>(`providers.${provider}.apiKey`) || oldApiKey || '';
  } else {
    // 获取提供商的预设配置
    const providerPresets = getProviderPresets();
    const preset = providerPresets[provider];

    if (!preset) {
      console.warn(`未找到提供商 ${provider} 的预设配置`);
    }
    apiUrl = config.get<string>(`providers.${provider}.url`) || preset?.url || '';
    model = config.get<string>(`providers.${provider}.model`) || preset?.model || '';
    protocol = config.get<string>(`providers.${provider}.protocol`) || preset?.protocol || 'openai';
    apiKey = config.get<string>(`providers.${provider}.apiKey`) || '';
  }

  // 从配置中获取提示词模板和系统指令
  const promptTemplate = config.get<string>('llm.prompt') || `请根据以下Git变更生成一句话提交信息，格式为<type>: <description>：\${diff}`;
  const system = config.get<string>('llm.system') || `请用一句话描述这次代码变更的主要内容，格式为<type>: <description>`

  // 替换模板变量
  const prompt = promptTemplate
    .replace(/\$\{files\}/g, stagedFiles.join('\n'))
    .replace(/\$\{diff\}/g, diffContent);

  // 解析URL（更健壮地处理用户自定义端点）
  if (!apiUrl) {
    console.warn(`[committer] llm apiUrl 为空，provider=${provider}，请检查扩展配置`);
  }
  const parsedUrl = url.parse(apiUrl || '');
  const isHttps = parsedUrl.protocol === 'https:';
  const parsedHostname = parsedUrl.hostname || '';

  // 获取匹配的服务配置（先尝试用解析到的hostname查找）
  let serviceConfig = modelServices.find(service => service.hostname === parsedHostname);
  if (!serviceConfig) {
    let serviceName = '';
    switch (protocol) {
      case 'ollama':
        serviceName = 'ollama';
        break;
      case 'openai':
      default:
        serviceName = 'openai';
        break;
    }
    serviceConfig = modelServices.find(service => service.name === serviceName);
  }
  if (!serviceConfig) {
    throw new Error(`未找到匹配的LLM服务配置: ${parsedHostname || apiUrl}`);
  }

  // 最终使用解析到的hostname，若为空则回退到serviceConfig中的hostname
  const hostname = parsedHostname || serviceConfig.hostname || 'localhost';
  const port = parsedUrl.port ? parseInt(parsedUrl.port, 10) : (isHttps ? 443 : 80);

  // 计算请求的 path，避免重复拼接或丢失斜杠
  const basePath = parsedUrl.pathname || parsedUrl.path || '';
  const suffix = serviceConfig.apiSuffix || '';
  let path = basePath || '';
  if (suffix) {
    if (!path.endsWith(suffix)) {
      // 处理斜杠，确保不会出现 // 或 缺少 /
      if (path.endsWith('/') && suffix.startsWith('/')) {
        path = path.slice(0, -1) + suffix;
      } else if (!path.endsWith('/') && !suffix.startsWith('/')) {
        path = path + '/' + suffix;
      } else {
        path = path + suffix;
      }
    }
  }
  if (!path) path = '/';
  console.log(`[committer] resolved apiUrl=${apiUrl} host=${hostname} port=${port} path=${path}`);
  let requestData = {};
  switch (serviceConfig.name.toLowerCase()) {
    case "aliyun":
      requestData = {
        model: model,
        messages: [
          {
            role: 'system',
            content: system
          },
          {
            role: 'user',
            content: prompt
          }
        ],
        max_tokens: maxTokens,
        stream: true
      };
      break;
    case "anthropic":
      requestData = {
        model: model,
        messages: [
          {
            role: 'user',
            content: prompt
          }
        ],
        system: system,
        max_tokens: maxTokens,
        temperature: temperature,
        stream: true
      };
      break;
    case "tencent":
      requestData = {
        model: model,
        messages: [
          {
            role: 'system',
            content: system
          },
          {
            role: 'user',
            content: prompt
          }
        ],
        temperature: temperature,
        enable_enhancement: false,
        top_p: topP,
        max_tokens: maxTokens,
        stream: true
      };
      break;
    case "deepseek":
      requestData = {
        model: model,
        messages: [
          {
            role: 'system',
            content: system
          },
          {
            role: 'user',
            content: prompt
          }
        ],
        temperature: temperature,
        top_p: topP,
        max_tokens: maxTokens,
        stream: true
      };
      break;
    case "siliconflow":
      requestData = {
        model: model,
        messages: [
          {
            role: 'system',
            content: system
          },
          {
            role: 'user',
            content: prompt
          }
        ],
        temperature: temperature,
        top_p: topP,
        max_tokens: maxTokens,
        stream: true
      };
      break;
      case "openai":
        requestData = {
          model: model,
          messages: [
            {
              role: 'system',
              content: system
            },
            {
              role: 'user',
              content: prompt
            }
          ],
          temperature: temperature,
          top_p: topP,
          max_tokens: maxTokens,
          stream: true
        };
        break;
      case "ollama":
        requestData = {
          model: model,
          system: system,
          prompt: prompt,
          temperature: temperature,
          top_p: topP,
          max_tokens: maxTokens,
          stream: true
        };
        break;
  }
  
  console.log('[committer] requestData:', requestData);

  // 创建请求选项
  const options = {
    hostname: hostname,
    port: port,
    path: path,
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    }
  };
  options.headers = serviceConfig.headers;
  if (apiKey && serviceConfig.AuthKey && typeof serviceConfig.AuthKey === 'string' && options.headers[serviceConfig.AuthKey as keyof typeof options.headers]) {
    const authKey = serviceConfig.AuthKey as keyof typeof options.headers;
    if (options.headers[authKey]) {
      options.headers[authKey] = options.headers[authKey] + apiKey;
    }
  }
  let optionsStr = JSON.stringify(options);
  console.log(`[committer]调用LLM API请求: ${optionsStr}`+'\n');

  return new Promise((resolve, reject) => {
    // 选择http或https模块
    const requester = isHttps ? https : http;
    
    const req = requester.request(options, (res) => {
      let data = '';
      
      let generatedText = '';
      let generatedThinking = '';
      let isThinking = false;

      res.on('data', (chunk) => {
        const lines = chunk.toString().split('\n').filter((line: string) => line.trim());
        
        for (const line of lines) {
          try {
            // 去除JSON字符串前的所有字符，只保留从{开始的部分
            const jsonStartIndex = line.indexOf('{');
            if (jsonStartIndex === -1) {
              continue;
            }
            const jsonStr = line.substring(jsonStartIndex);
            const response = JSON.parse(jsonStr);
            
            if (!serviceConfig) {
              throw new Error('未找到匹配的LLM服务配置');
            }
            
            switch (serviceConfig.protocol) {
              case "openai":
                if (response.choices){
                  // 处理流式delta（已有逻辑）
                  if(response.choices[0]?.delta?.content) {
                    let content = response.choices[0].delta.content;
                    if (isThinking) {
                      content = content.replace(/^\n+/, ' ');
                      generatedThinking += content;
                    } else {
                      if (generatedText === '') {
                        content = content.replace(/^\n+/, '');
                      } else {
                        content = content.replace(/^```[a-z0-9]+\n/g, '').replace(/```/g, '');
                      }
                      generatedText += content;
                    }
                    if (generatedText.match(/^<think>/)) {
                      generatedThinking = generatedText.replace(/^<think>/, '');
                      generatedText = '';
                      isThinking = true;
                    }
                    const thinkEndMatch = generatedThinking.match(/<\/think>(.*)$/);
                    if (thinkEndMatch) {
                      isThinking = false;
                      generatedText = thinkEndMatch[1].replace(/^\n+/, '');
                      generatedThinking = generatedThinking.replace(/<\/think>.*$/, '');
                    }
                    if (isThinking) {
                      statusBarMessage.text = generatedThinking;
                      statusBarMessage.show();
                      if (generatedThinking.length > 30) {
                        generatedThinking = '';
                      }
                    } else {
                      inputBox.value = generatedText;
                    }
                  }
                  if(response.choices[0]?.delta?.reasoning_content) {
                    if (generatedThinking.length > 30) {
                      generatedThinking = '';
                    }
                    generatedThinking += response.choices[0].delta.reasoning_content.replace(/\n/g, ' ');
                    statusBarMessage.text = generatedThinking;
                    statusBarMessage.show();
                  }

                  // 处理非流式（一次性）响应：OpenAI Chat Completions 或 Completions
                  const choice = response.choices[0];
                  if (choice) {
                    // Chat completion: message.content 可能为字符串或对象（含 parts）
                    let fullContent = '';
                    if (choice.message && choice.message.content) {
                      const mc = choice.message.content as any;
                      if (typeof mc === 'string') {
                        fullContent = mc;
                      } else if (Array.isArray(mc.parts)) {
                        fullContent = mc.parts.join('');
                      } else {
                        try {
                          fullContent = JSON.stringify(mc);
                        } catch (e) {
                          fullContent = String(mc);
                        }
                      }
                    } else if (choice.text) {
                      // Completion API 返回的 text 字段
                      fullContent = choice.text as string;
                    }

                    if (fullContent) {
                      // 清理代码块并处理 <think> 标签（与流式逻辑保持一致）
                      fullContent = fullContent.replace(/^```[a-z0-9]+\n/g, '').replace(/```/g, '');
                      if (fullContent.match(/^<think>/)) {
                        generatedThinking = fullContent.replace(/^<think>/, '');
                        generatedText = '';
                        isThinking = true;
                      } else {
                        generatedText += fullContent;
                      }

                      if (isThinking) {
                        statusBarMessage.text = generatedThinking;
                        statusBarMessage.show();
                      } else {
                        inputBox.value = generatedText;
                      }
                    }
                  }
                }
                break;
              case "ollama":
              default:
                if (response.response) {
                  let content = response.response;
                  if (isThinking) {
                    content = content.replace(/^\n+/, ' ')
                    generatedThinking += content;
                  } else {
                    if (generatedText === '') {
                      content = content.replace(/^\n+/, '');
                    }
                    generatedText += content;
                  }
                  if (generatedText.match(/^<think>/)) {
                    generatedThinking = generatedText.replace(/^<think>/, '');
                    generatedText = '';
                    isThinking = true;
                  }
                  const thinkEndMatch = generatedThinking.match(/<\/think>(.*)$/);
                  if (thinkEndMatch) {
                    isThinking = false;
                    generatedText = thinkEndMatch[1].replace(/^\n+/, '');
                    generatedThinking = generatedThinking.replace(/<\/think>.*$/, '');
                  }
                  if (isThinking) {
                    statusBarMessage.text = generatedThinking;
                    statusBarMessage.show();
                    if (generatedThinking.length > 30) {
                      generatedThinking = '';
                    }
                  } else {
                    inputBox.value = generatedText;
                  }
                  console.log('[committer]', generatedText, generatedThinking)
                }
                break;
            }
          } catch (error) {
            // 如果解析JSON失败，可能是因为接收到了不完整的数据块
            console.log('解析数据块失败，跳过:', error);
          }
        }
      });
      
      res.on('end', () => {
        if (generatedText) {
          resolve(generatedText.trim().replace(/^```[a-zA-Z0-9]*\n|```/g, ''));
        } else {
          reject(new Error('未收到有效的响应数据'));
        }
      });
    });
    
    req.on('error', (error) => {
      reject(new Error(`API请求错误: ${error.message}`));
    });
    
    // 发送请求数据
    req.write(JSON.stringify(requestData));
    req.end();
  });
}

/**
 * 根据暂存文件和diff内容生成commit message
 */
async function generateCommitMessage(stagedFiles: string[], diffContent: string, inputBox: any, statusBarMessage: vscode.StatusBarItem): Promise<string> {
  try {
    // 调用LLM API生成commit message
    return await callLLMAPI(stagedFiles, diffContent, inputBox, statusBarMessage);
  } catch (error) {
    console.error('调用LLM API失败:', error);
    const errorMessage = error instanceof Error ? error.message : String(error);
    vscode.window.showErrorMessage(`调用LLM API失败: ${errorMessage}`);
    
    // 如果API调用失败，回退到本地生成逻辑
    return generateLocalCommitMessage(stagedFiles, diffContent);
  }
}

/**
 * 本地生成commit message的备用方法
 */
function generateLocalCommitMessage(stagedFiles: string[], diffContent: string): string {
  // 检查是否有新增文件
  const newFiles = stagedFiles.filter(file => file.startsWith('A '));
  // 检查是否有修改文件
  const modifiedFiles = stagedFiles.filter(file => file.startsWith('M '));
  // 检查是否有删除文件
  const deletedFiles = stagedFiles.filter(file => file.startsWith('D '));

  let prefix = '';
  
  // 根据变更类型确定前缀
  if (newFiles.length > 0 && modifiedFiles.length === 0 && deletedFiles.length === 0) {
    prefix = 'feat: ';
  } else if (deletedFiles.length > 0 && newFiles.length === 0 && modifiedFiles.length === 0) {
    prefix = 'remove: ';
  } else if (modifiedFiles.length > 0) {
    // 检查是否包含测试文件
    const isTestChange = modifiedFiles.some(file => 
      file.includes('test') || file.includes('spec')
    );
    
    if (isTestChange) {
      prefix = 'test: ';
    } else {
      // 检查是否是bug修复
      const isBugFix = diffContent.includes('fix') || 
                      diffContent.includes('bug') || 
                      diffContent.includes('issue');
      
      if (isBugFix) {
        prefix = 'fix: ';
      } else {
        prefix = 'feat: ';
      }
    }
  } else if (deletedFiles.length > 0) {
    // 如果有删除文件但同时有其他类型的变更，优先考虑是否为删除操作
    prefix = 'remove: ';
  } else {
    prefix = 'chore: ';
  }

  // 生成简单的描述
  let description = '';
  
  if (stagedFiles.length === 1) {
    // 如果只有一个文件，使用文件名作为描述的一部分
    const fileName = stagedFiles[0].split(' ').pop() || '';
    const fileNameWithoutExt = fileName.split('.').shift() || '';
    
    if (newFiles.length === 1) {
      description = `添加${fileNameWithoutExt}功能`;
    } else if (modifiedFiles.length === 1) {
      description = `更新${fileNameWithoutExt}功能`;
    } else if (deletedFiles.length === 1) {
      // 对删除文件提供更具体的描述
      description = `删除${fileNameWithoutExt}`;
    }
  } else {
    // 多个文件的情况
    if (newFiles.length > 0 && modifiedFiles.length === 0 && deletedFiles.length === 0) {
      description = `添加新功能，涉及${newFiles.length}个文件`;
    } else if (deletedFiles.length > 0 && newFiles.length === 0 && modifiedFiles.length === 0) {
      description = `删除文件，共${deletedFiles.length}个`;
    } else if (modifiedFiles.length > 0) {
      description = `更新功能，涉及${modifiedFiles.length}个文件`;
    } else {
      description = `代码变更，涉及${stagedFiles.length}个文件`;
    }
  }

  return `${prefix}${description}`;
}

/**
 * 获取各个提供商的预设配置
 */
function getProviderPresets(): Record<string, any> {
  return {
    'aliyun': {
      url: 'https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions',
      model: 'deepseek-r1-distill-llama-70b',
      protocol: 'openai'
    },
    'openai': {
      url: 'https://api.openai.com/v1/chat/completions',
      model: 'gpt-4o-mini',
      protocol: 'openai'
    },
    'ollama': {
      url: 'http://localhost:11434/api/generate',
      model: 'deepseek-r1:7b',
      protocol: 'ollama'
    },
    'deepseek': {
      url: 'https://api.deepseek.com/v1/chat/completions',
      model: 'deepseek-chat',
      protocol: 'openai'
    },
    'anthropic': {
      url: 'https://api.anthropic.com/v1/messages',
      model: 'claude-3-haiku-20240307',
      protocol: 'anthropic'
    },
    'tencent': {
      url: 'https://api.hunyuan.cloud.tencent.com/v1/chat/completions',
      model: 'hunyuan-lite',
      protocol: 'openai'
    },
    'siliconflow': {
      url: 'https://api.siliconflow.cn/v1/chat/completions',
      model: 'deepseek-ai/DeepSeek-V3',
      protocol: 'openai'
    },
    'volcengine': {
      url: 'https://ark.cn-beijing.volces.com/api/v3/chat/completions',
      model: 'deepseek-v3-250324',
      protocol: 'openai'
    },
    'custom': {
      url: '',
      model: '',
      protocol: 'openai'
    }
  };
}

export function deactivate() {}