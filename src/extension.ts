import * as vscode from 'vscode';
import { simpleGit, SimpleGit } from 'simple-git';
import * as https from 'https';
import * as http from 'http';
import * as url from 'url';
import { log } from 'console';

export function activate(context: vscode.ExtensionContext) {
  console.log('插件 "vscode-git-commit-message-generator" 已激活');

  // 注册命令
  let disposable = vscode.commands.registerCommand('vscode-git-commit-message-generator.generateCommitMessage', async () => {
    try {
      // 获取当前工作区
      const workspaceFolders = vscode.workspace.workspaceFolders;
      if (!workspaceFolders || workspaceFolders.length === 0) {
        vscode.window.showErrorMessage('没有打开的工作区');
        return;
      }

      const rootPath = workspaceFolders[0].uri.fsPath;
      const git: SimpleGit = simpleGit(rootPath);

      // 检查是否有staged文件
      const status = await git.status();
      if (status.staged.length === 0) {
        vscode.window.showWarningMessage('没有暂存的文件，请先添加文件到暂存区');
        return;
      }

      // 获取暂存区的文件变更
      const stagedFiles = status.staged;
      vscode.window.showInformationMessage(`找到 ${stagedFiles.length} 个暂存的文件`);

      // 获取每个文件的diff
      let allDiffs = '';
      for (const file of stagedFiles) {
        try {
          const diff = await git.diff(['--cached', file]);
          allDiffs += `\n文件: ${file}\n${diff}\n`;
        } catch (error) {
          console.error(`获取文件 ${file} 的diff失败:`, error);
        }
      }

      // 根据变更内容生成commit message
      const commitMessage = await generateCommitMessage(stagedFiles, allDiffs);

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

      // 获取当前仓库
      const repositories = api.repositories;
      if (repositories.length === 0) {
        vscode.window.showErrorMessage('没有找到Git仓库');
        return;
      }

      // 直接设置commit message，不再弹出确认输入框
      repositories[0].inputBox.value = commitMessage;
      vscode.window.showInformationMessage('已设置commit message');
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
async function callLLMAPI(stagedFiles: string[], diffContent: string): Promise<string> {
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
      apiSuffix: '/messages',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': '',
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
  const config = vscode.workspace.getConfiguration('vscode-git-commit-message-generator.llm');
  const apiUrl = config.get<string>('url') || 'http://ollama.e.weibo.com';
  const model = config.get<string>('model') || 'QwQ-32B-AWQ';
  const temperature = config.get<number>('temperature') || 0.7;
  const topP = config.get<number>('topP') || 1;
  const protocol = config.get<string>('protocol') || 'ollama';

  // 从配置中获取提示词模板和系统指令
  const promptTemplate = config.get<string>('prompt') || `请根据以下Git变更生成一句话提交信息，格式为<type>: <description>：\${diff}`;
  const system = config.get<string>('system') || `请用一句话描述这次代码变更的主要内容，格式为<type>: <description>`

  // 替换模板变量
  const prompt = promptTemplate
    .replace(/\$\{files\}/g, stagedFiles.join('\n'))
    .replace(/\$\{diff\}/g, diffContent);

  // 解析URL
  const parsedUrl = url.parse(apiUrl);
  const isHttps = parsedUrl.protocol === 'https:';
  const hostname = parsedUrl.hostname || 'localhost';
  const port = parsedUrl.port ? parseInt(parsedUrl.port, 10) : (isHttps ? 443 : 80);

  // 获取匹配的服务配置
  let serviceConfig = modelServices.find(service => service.hostname === hostname);
  if (!serviceConfig) {
    let serviceName = '';
    switch (protocol) {
      case 'openai':
        serviceName = 'openai';
        break;
      case 'anthropic':
        serviceName = 'anthropic';
        break;
      case 'ollama':
      default:
        serviceName = 'ollama';
        break;
    }
    serviceConfig = modelServices.find(service => service.name === serviceName);
  }
  if (!serviceConfig) {
    throw new Error(`未找到匹配的LLM服务配置: ${hostname}`);
  }

  const path = parsedUrl.pathname?.match(new RegExp(`${serviceConfig.apiSuffix}$`)) ? parsedUrl.pathname : `${parsedUrl.path}${serviceConfig.apiSuffix}`;
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
        max_tokens: 1024,
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
          stream: true
        };
        break;
  }
  
  console.log('[Committer] requestData:', requestData);

  // 获取API密钥
  const apiKey = config.get<string>('apiKey') || '';

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
  console.log(`[Committer]调用LLM API请求: ${optionsStr}`+'\n');

  return new Promise((resolve, reject) => {
    // 选择http或https模块
    const requester = isHttps ? https : http;
    
    const req = requester.request(options, (res) => {
      let data = '';
      
      let generatedText = '';
      
      res.on('data', (chunk) => {
        const lines = chunk.toString().split('\n').filter((line: string) => line.trim());
        
        for (const line of lines) {
          try {
            if (line === '[DONE]') {
              resolve(generatedText.trim());
              return;
            }
            
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
                if (response.choices && response.choices[0]?.delta?.content) {
                  generatedText += response.choices[0].delta.content;
                }
                break;
              case "anthropic":
                if (response.type === 'content_block_delta' && response.delta?.text) {
                  generatedText += response.delta.text;
                }
                break;
              case "openrouter":
                if (response.completion) {
                  generatedText += response.completion;
                }
                break;
              case "ollama":
              default:
                if (response.response) {
                  generatedText += response.response;
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
          resolve(generatedText.trim());
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
async function generateCommitMessage(stagedFiles: string[], diffContent: string): Promise<string> {
  try {
    // 调用LLM API生成commit message
    return await callLLMAPI(stagedFiles, diffContent);
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
    prefix = 'chore: ';
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
      description = `移除${fileNameWithoutExt}功能`;
    }
  } else {
    // 多个文件的情况
    if (newFiles.length > 0 && modifiedFiles.length === 0 && deletedFiles.length === 0) {
      description = `添加新功能，涉及${newFiles.length}个文件`;
    } else if (deletedFiles.length > 0 && newFiles.length === 0 && modifiedFiles.length === 0) {
      description = `移除功能，涉及${deletedFiles.length}个文件`;
    } else if (modifiedFiles.length > 0) {
      description = `更新功能，涉及${modifiedFiles.length}个文件`;
    } else {
      description = `代码变更，涉及${stagedFiles.length}个文件`;
    }
  }

  return `${prefix}${description}`;
}

export function deactivate() {}