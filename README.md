# Crush · 面向开发与服务器运维的终端 AI 助手

[![Release](https://img.shields.io/github/v/release/clvr024/crush)](https://github.com/clvr024/crush/releases/latest)
[![Build](https://github.com/clvr024/crush/actions/workflows/build.yml/badge.svg)](https://github.com/clvr024/crush/actions/workflows/build.yml)
[![Sync upstream and release](https://github.com/clvr024/crush/actions/workflows/upstream-sync.yml/badge.svg)](https://github.com/clvr024/crush/actions/workflows/upstream-sync.yml)

这是 [charmbracelet/crush](https://github.com/charmbracelet/crush) 的个人维护分支，围绕日常编程和服务器使用中的实际问题做小幅改进：减少重复探索，让任务进度更清楚，让终端界面更多地展示问题和结论。改动尽量保持局部，便于持续合并上游更新。

保留上游的多模型切换、项目会话、LSP 代码上下文、MCP 扩展和技能支持，可连接 Gemini、OpenAI、Anthropic 等服务以及兼容 API。

[下载安装](#下载安装) · [快速开始](#快速开始) · [权限模式](#权限模式) · [配置参考](docs/config/README.md) · [构建与发布](#构建与发布)

## 本分支的主要改进

| 使用中的问题 | 本分支的处理方式 |
| --- | --- |
| 反复读同一个文件，或把文件拆成很多小段读取 | 文件工具默认读取 1,000 行，内容上限为 200 KB；提示模型先定位再读取大文件，并检测重复搜索、重复读取及高度重叠的读取，必要时返回调整方向的提示。 |
| 已有发现却迟迟不反馈 | 加强沟通提示，要求模型及时说明线索、可能原因、阻塞点，以及耗时操作的目的和状态。 |
| 命令转入后台后缺少跟进 | Bash 默认在运行 60 秒后转为后台任务，返回任务 ID 和后续查询指引，提醒模型收集输出和最终退出状态。 |
| 工具输出占满对话 | 工具调用默认显示一行摘要，连续工具之间去掉空行，需要时展开详情。 |
| 长会话上下滚动卡顿 | 缓存消息起始行位置，减少滚动时的重复遍历。 |
| 命令结果状态不准确、短任务额外等待 | 修正 Bash 失败状态和后台任务完成状态；显式后台执行的短命令完成后立即返回结果。 |

重复调用检测和工具状态处理由代码实现；进度反馈与后续操作也依赖所选模型对提示的遵循程度。

## 下载安装

从本仓库的 **[最新 Release](https://github.com/clvr024/crush/releases/latest)** 下载对应系统和架构的安装包：

| 系统 | 架构 | 格式 |
| --- | --- | --- |
| Linux | amd64 / arm64 | `.tar.gz` |
| macOS | amd64 / arm64 | `.tar.gz` |
| Windows | amd64 / arm64 | `.zip` |

Release 同时提供 `checksums.txt`，可用于校验下载文件。常见 Intel / AMD 电脑选择 `amd64`，Apple Silicon 和 ARM 服务器选择 `arm64`。

Linux / macOS 解压后，在解压目录执行：

```bash
chmod +x crush
./crush --version
./crush
```

如需在任意目录使用，可将可执行文件放入 `PATH`，例如：

```bash
sudo install -m 0755 crush /usr/local/bin/crush
```

Windows 解压后，在 PowerShell 中进入解压目录执行：

```powershell
.\crush.exe --version
.\crush.exe
```

也可以把解压目录加入 `PATH`，之后直接使用 `crush`。下方上游参考中的 Homebrew、npm、Winget、Scoop 等命令安装的是上游版本；获取本分支改动请使用本仓库的 Release。

## 快速开始

在项目目录启动 `crush`，按界面提示配置服务商和 API Key；也可使用 `Ctrl+L` 选择或切换模型。

```bash
# 在当前目录开始会话。
crush

# 指定项目或运维工作目录。
crush --cwd /path/to/project

# 继续最近一次会话。
crush --continue

# 非交互运行，适合脚本或没有交互终端的环境。
crush run "检查当前目录的项目结构，说明主要入口"
```

工具调用默认折叠为一行，点击可展开或收起。键盘操作时，按 `Tab` 聚焦对话区域，用 `Shift+↑/↓` 在消息间移动，按 `Space` 展开或收起；`Ctrl+P` 打开命令菜单。

耗时命令转入后台表示命令仍在执行。工具会返回后台任务 ID，模型可通过 `job_output` 查询输出与退出状态；仅在需要终止时使用 `job_kill`。

## 权限模式

| 启动方式 | 行为 |
| --- | --- |
| `crush` | 普通模式：安全的读取可直接执行，其他操作按权限规则请求确认。 |
| `crush -y` | YOLO 模式：自动批准权限请求，保留执行时的命令阻止列表。 |
| `crush -yy` | Sysadmin 模式：自动批准权限请求，同时关闭执行时的命令阻止列表。 |

`-y` 也会自动批准在权限检查阶段识别出的危险命令，不能视为安全沙箱。`-yy` 适合你明确授权操作的可信环境，它不会提升系统权限，命令仍以当前操作系统用户身份执行。

客户端连接已有服务端工作区时，权限模式由首次创建工作区的设置决定，后续客户端的启动参数不一定会改变该模式。

## 配置与扩展

推荐使用 `crushrc`，通过 Bash 语法配置模型、服务商、MCP、LSP、权限和 Hooks。项目配置可以使用 `.crushrc` 或 `crushrc`；默认全局配置位置为：

- Linux / macOS：`~/.config/crush/crushrc`
- Windows：`%USERPROFILE%\.config\crush\crushrc`

设置了 `XDG_CONFIG_HOME` 时，配置路径随之调整。旧的 `crush.json` 仍然受支持，但已弃用，推荐逐步迁移到 `crushrc`。运行 `crush dirs` 可查看实际使用的配置和数据目录。

- [完整配置参考](docs/config/README.md)：服务商、模型、MCP、LSP、权限及旧配置迁移。
- [Hooks 文档](docs/hooks/README.md)：工具执行前的自定义检查与处理。
- 项目规则可放在 `AGENTS.md`、`CRUSH.md`、`CLAUDE.md` 或 `GEMINI.md` 中，作为模型的项目上下文。

## 构建与发布

向 `main` 推送或提交 PR 后，[Build 工作流](https://github.com/clvr024/crush/actions/workflows/build.yml)会在 Linux、macOS 和 Windows 上执行构建与测试。**推送代码本身不会创建 Release。**

需要生成安装包时，在 GitHub Actions 中手动运行 [Sync upstream and release](https://github.com/clvr024/crush/actions/workflows/upstream-sync.yml)。该流程会检查并合并上游、运行测试，通过后推送版本标签并发布安装包；手动运行时，即使没有上游更新也会请求发布。版本号采用 `v上游版本+fork.流程运行号` 的形式。

该流程也按计划每天检查上游（UTC 02:17，北京时间 10:17）。合并时保留本分支的工作流配置；其他文件发生冲突会停止流程，需要手动处理后再发布。

## 上游与许可证

本项目基于 [Charm 的 Crush](https://github.com/charmbracelet/crush)，本仓库提供的安装包属于个人维护分支。许可证沿用 [FSL-1.1-MIT](LICENSE.md)。以下保留上游原始说明，便于查阅完整功能和使用参考。

<details>
<summary><strong>展开上游完整说明与配置参考</strong></summary>

# Crush

<p align="center">
    <a href="https://stuff.charm.sh/crush/charm-crush.png"><img width="450" alt="Charm Crush Logo" src="https://github.com/user-attachments/assets/cf8ca3ce-8b02-43f0-9d0f-5a331488da4b" /></a><br />
    <a href="https://github.com/charmbracelet/crush/releases"><img src="https://img.shields.io/github/release/charmbracelet/crush" alt="Latest Release"></a>
    <a href="https://github.com/charmbracelet/crush/actions"><img src="https://github.com/charmbracelet/crush/actions/workflows/build.yml/badge.svg" alt="Build Status"></a>
</p>

<p align="center">Your new coding bestie, now available in your favourite terminal.<br />Your tools, your code, and your workflows, wired into your LLM of choice.</p>
<p align="center">终端里的编程新搭档，<br />无缝接入你的工具、代码与工作流，全面兼容主流 LLM 模型。</p>

<p align="center"><img width="800" alt="Crush Demo" src="https://github.com/user-attachments/assets/58280caf-851b-470a-b6f7-d5c4ea8a1968" /></p>

## Features

- **Multi-Model:** choose from a wide range of LLMs or add your own via OpenAI- or Anthropic-compatible APIs
- **Flexible:** switch LLMs mid-session while preserving context
- **Session-Based:** maintain multiple work sessions and contexts per project
- **LSP-Enhanced:** Crush uses LSPs for additional context, just like you do
- **Extensible:** add capabilities via MCPs (`http`, `stdio`, and `sse`)
- **Works Everywhere:** first-class support in every terminal on macOS, Linux, Windows (PowerShell and WSL), Android, FreeBSD, OpenBSD, and NetBSD
- **Industrial Grade:** built on the Charm ecosystem, powering 25k+ applications, from leading open source projects to business-critical infrastructure

## Installation

Use a package manager:

```bash
# Homebrew
brew install charmbracelet/tap/crush

# NPM (macOS, Linux, and Windows)
npm install -g @charmland/crush

# Arch Linux (btw)
yay -S crush

# Nix
nix run github:numtide/nix-ai-tools#crush

# FreeBSD
pkg install crush
```

Windows users:

```bash
# Winget
winget install charmbracelet.crush

# Scoop
scoop bucket add charm https://github.com/charmbracelet/scoop-bucket.git
scoop install crush
```

<details>
<summary><strong>Nix (NUR)</strong></summary>

Crush is available via the official Charm [NUR](https://github.com/nix-community/NUR) in `nur.repos.charmbracelet.crush`, which is the most up-to-date way to get Crush in Nix.

You can also try out Crush via the NUR with `nix-shell`:

```bash
# Add the NUR channel.
nix-channel --add https://github.com/nix-community/NUR/archive/main.tar.gz nur
nix-channel --update

# Get Crush in a Nix shell.
nix-shell -p '(import <nur> { pkgs = import <nixpkgs> {}; }).repos.charmbracelet.crush'
```

### NixOS & Home Manager Module Usage via NUR

Crush provides NixOS and Home Manager modules via NUR.
You can use these modules directly in your flake by importing them from NUR. Since it auto detects whether it's a home manager or nixos context you can use the import the exact same way :)

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nur.url = "github:nix-community/NUR";
  };

  outputs = { self, nixpkgs, nur, ... }: {
    nixosConfigurations.your-hostname = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        nur.modules.nixos.default
        nur.repos.charmbracelet.modules.crush
        {
          programs.crush = {
            enable = true;
            settings = {
              providers = {
                openai = {
                  id = "openai";
                  name = "OpenAI";
                  base_url = "https://api.openai.com/v1";
                  type = "openai";
                  api_key = "sk-fake123456789abcdef...";
                  models = [
                    {
                      id = "gpt-5.6-sol";
                      name = "GPT-5.6 Sol";
                    }
                  ];
                };
              };
              lsp = {
                go = { command = "gopls"; enabled = true; };
                nix = { command = "nil"; enabled = true; };
              };
              options = {
                context_paths = [ "/etc/nixos/configuration.nix" ];
                tui = { compact_mode = true; };
                debug = false;
              };
            };
          };
        }
      ];
    };
  };
}
```

</details>

<details>
<summary><strong>Debian/Ubuntu</strong></summary>

```bash
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
sudo apt update && sudo apt install crush
```

</details>

<details>
<summary><strong>Fedora/RHEL</strong></summary>

```bash
echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | sudo tee /etc/yum.repos.d/charm.repo
sudo yum install crush
```

</details>

Or, download it:

- [Packages][releases] are available in Debian and RPM formats
- [Binaries][releases] are available for Linux, macOS, Windows, FreeBSD, OpenBSD, and NetBSD

[releases]: https://github.com/charmbracelet/crush/releases

Or just install it with Go:

```
go install github.com/charmbracelet/crush@latest
```

On illumos (OpenIndiana, OmniOS), the command above works as-is. Only native
OS notifications are unavailable there; terminal-based notifications (OSC) and
the terminal bell still work. On Oracle Solaris, add `-tags sqlite3_dotlk` so
the local database uses dot-file locking:

```
go install -tags sqlite3_dotlk github.com/charmbracelet/crush@latest
```

> [!WARNING]
> Productivity may increase when using Crush and you may find yourself nerd
> sniped when first using the application. If the symptoms persist, join the
> [Slack][slack] or [Discord][discord] and nerd snipe the rest of us.

## Getting Started

The quickest way to get started is to choose a [Hyper][hyper] model from the
little model picker. Follow the steps to authenticate and you're ready to go.

<p><a href="https://hyper.charm.land"><img width="340" height="200" alt="Charm Hyper" src="https://github.com/user-attachments/assets/50875289-7992-454d-9f14-9f790413fb5e" /></a></p>

[Hyper][hyper], from Charm, is the official model provider for Crush. It’s
built  on open-weight models and is optimized for Crush, with an emphasis
on high-quality inference. There are subscription and prepaid options, including zero data retention
(ZDR). It's also GDPR compliant!

[More on Hyper][hyper].


## API Keys

You can also use Crush with many other providers such as Anthopic, OpenAI,
Gemini, OpenRouter and so on. Press <kbd>ctrl+l</kbd> to open the model picker,
choose the provider of your choice, and paste your API key.

That said, you can also set environment variables for preferred providers:

| Environment Variable        | Provider                                           |
| --------------------------- | -------------------------------------------------- |
| `HYPER_API_KEY`             | [Charm Hyper][hyper]                               |
| `ANTHROPIC_API_KEY`         | Anthropic                                          |
| `OPENAI_API_KEY`            | OpenAI                                             |
| `VERCEL_API_KEY`            | Vercel AI Gateway                                  |
| `GEMINI_API_KEY`            | Google Gemini                                      |
| `ZAI_API_KEY`               | Z.ai                                               |
| `MINIMAX_API_KEY`           | MiniMax                                            |
| `SYNTHETIC_API_KEY`         | Synthetic                                          |
| `HF_TOKEN`                  | Hugging Face Inference                             |
| `CEREBRAS_API_KEY`          | Cerebras                                           |
| `OPENROUTER_API_KEY`        | OpenRouter                                         |
| `IONET_API_KEY`             | io.net                                             |
| `ALIBABA_SINGAPORE_API_KEY` | Alibaba (Singapore)                                |
| `ALIBABA_US_API_KEY`        | Alibaba (United States)                            |
| `GROQ_API_KEY`              | Groq                                               |
| `AVIAN_API_KEY`             | Avian                                              |
| `OPENCODE_API_KEY`          | OpenCode Zen & Go                                  |
| `VERTEXAI_PROJECT`          | Google Cloud VertexAI (Gemini)                     |
| `VERTEXAI_LOCATION`         | Google Cloud VertexAI (Gemini)                     |
| `AWS_ACCESS_KEY_ID`         | Amazon Bedrock (Claude)                            |
| `AWS_SECRET_ACCESS_KEY`     | Amazon Bedrock (Claude)                            |
| `AWS_REGION`                | Amazon Bedrock (Claude)                            |
| `AWS_PROFILE`               | Amazon Bedrock (Custom Profile)                    |
| `AWS_BEARER_TOKEN_BEDROCK`  | Amazon Bedrock                                     |
| `AZURE_OPENAI_API_ENDPOINT` | Azure OpenAI models                                |
| `AZURE_OPENAI_API_KEY`      | Azure OpenAI models (optional when using Entra ID) |
| `AZURE_OPENAI_API_VERSION`  | Azure OpenAI models                                |
| `MOONSHOT_API_KEY`          | Moonshot                                           |
| `XAI_API_KEY`               | xAI                                                |

[hyper]: https://hyper.charm.land

Also note that Crush can support nearly any provider, including
[Local Models](#local-models). For more info see
[Custom Providers](#custom-providers) below.

### By the Way

Is there a provider you’d like to see in Crush? Is there an existing model that needs an update?

Crush’s default model listing is managed in [Catwalk](https://github.com/charmbracelet/catwalk), a community-supported, open source repository of Crush-compatible models, and you’re welcome to contribute.

<a href="https://github.com/charmbracelet/catwalk"><img width="174" height="174" alt="Catwalk Badge" src="https://github.com/user-attachments/assets/95b49515-fe82-4409-b10d-5beb0873787d" /></a>

## Configuration

> [!TIP]
> Crush ships with a builtin skill for configuring itself. Most of the time
> you can just tell what you want it to configure and it will get the job done.

Crush runs great with no configuration. That said, if you do need or want to
customize Crush, you can, with a `crushrc`.

A `crushrc` is just Bash with some Crush-specific builtins. It’s a lot like
a `.bashrc`, just for your Crush. Because Crush has a native, built-in Bash
interpreter, Bash-based config works identically across all platforms, including
Windows.

For example:

```bash
# Add Ollama.
provider add ollama --type ollama --base-url "http://localhost:11434/v1"

# Register a model on Ollama.
model add ollama/llama3.3 --name "Llama 3.3" --context-window 128000

# Auto-approve some tools.
permissions allow view edit

# Include some other file on a specific machine.
if [[ $HOSTNAME == "babysquid" ]]; then
    source ~/my-stuff/babysquid.sh
fi

# Add an MCP server, with a GitHub API token stored in 1Password.
mcp add github \
  --type http \
  --url "https://api.github.com/mcp/" \
  --header Authorization "Bearer $(op read 'op://my-secret-key')"
```

Configuration can be added either local to the project itself, or globally,
with the following priority:

| Priority | Unix-like                 | Windows                               |
| -------- | ------------------------- | ------------------------------------- |
| 1        | `./.crushrc`              | `.\.crushrc`                          |
| 2        | `./crushrc`               | `.\crushrc`                           |
| 3        | `~/.config/crush/crushrc` | `%USERPROFILE%\.config\crush\crushrc` |

(Crush respects the [XDG Base Directory Specification][xdg], so your paths
may differ depending on your `XDG_CONFIG_HOME` value. Data directories such as
`~/.local/share/crush` and `%LOCALAPPDATA%\crush` contain JSON state only; Crush
does not execute a `crushrc` from them.)

[xdg]: https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html

What about the old JSON format? It’s still supported, but it should be
considered deprecated. See: [the config docs](./docs/config/) for details.

> [!TIP]
> You can override the user and data config locations by setting:
>
> - `CRUSH_GLOBAL_CONFIG`
> - `CRUSH_GLOBAL_DATA`

As an additional note, Crush also stores ephemeral data, such as application
state, in one additional location. This is state and should not be edited by
hand, nor should it be considered configuration.

```bash
# Unix
$HOME/.local/share/crush/crush.json

# Windows
%LOCALAPPDATA%\crush\crush.json
```

#### A note on security

Both `crushrc` and `crush.json` are trusted code; `crushrc` runs in a full
shell, and any `$(...)` in `crush.json` runs at load time. Don't launch Crush
in a directory whose config you haven't reviewed, and don't randomly `source`
files from the internet into your config.

### Environment Variables

The top-level `env` field sets environment variables at startup, before
providers are configured. This is useful for variables that affect provider
authentication (e.g. the AWS SDK credential chain) without wrapping the
`crush` command in a shell script or exporting them in your shell profile:

```json
{
  "$schema": "https://charm.land/crush.json",
  "env": {
    "AWS_PROFILE": "my-sso-profile"
  }
}
```

Values support the same `$VAR` and `$(command)` expansion as other config
fields, so you can reference existing environment variables or shell out for
a value.

### Themes

Crush ships with built-in color themes.

#### Switching Themes

Open the command palette with `ctrl+p`, select **Themes**, and browse the
list. The UI previews each theme as you navigate, and pressing `enter`
confirms the selection. Press `esc` to cancel and revert.

#### Editing Themes

Open **Themes**, highlight the theme to customize, and press `ctrl+e`.
Changes preview live as you type. Press `enter` or `ctrl+s` to save, or `esc`
to cancel and revert. User themes are stored globally in the Crush config
directory under `themes/`.

You can also select a theme directly in your config with `active_theme`:

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "tui": {
      "active_theme": "gruvbox-dark"
    }
  }
}
```

Custom theme palettes are stored as JSON files in the global theme directory.
For example, `~/.config/crush/themes/my-theme.json`:

```json
{
  "base": "gruvbox-dark",
  "primary": "#ff6b6b",
  "bg_base": "#1a1a2e"
}
```

Select it by setting `active_theme` to `my-theme` or from the **Themes**
dialog.

#### Built-In Themes

| Theme             | Name                          |
| ----------------- | ----------------------------- |
| Charmtone Pantera | `charmtone-panther` (default) |
| Gruvbox Dark      | `gruvbox-dark`                |
| Keanu             | `keanu`                       |

### LSPs

Crush can use LSPs for additional context to help inform its decisions, just
like you would. LSPs can be added manually like so:

```bash
# crushrc

lsp add go --command "gopls" --env "GOTOOLCHAIN go1.24.5"
lsp add typescript --command "typescript-language-server" --args --stdio
lsp add nix --command "nil"
```

### MCPs

Crush also supports Model Context Protocol (MCP) servers through three transport
types: `stdio` for command-line servers, `http` for HTTP endpoints, and `sse`
for Server-Sent Events.

```bash
# crushrc

# Add a local MCP server that runs a Node.js script.
mcp add filesystem --command node --args /path/to/mcp-server.js \
  --timeout 10 --disabled-tools some-tool-name --env NODE_ENV production

# Add a GitHub MCP server that uses an API token.
mcp add github --type http --url https://api.github.com/mcp/ \
  --timeout 10 --header Authorization "Bearer $GH_PAT" \
  --disabled-tools create_issue --disabled-tools create_pull_request

# Add a streaming MCP server that uses SSE.
mcp add streaming-service --type sse --url "https://example.com/mcp/sse" \
  --timeout 10 --header API-Key "$API_KEY"
```

#### MCP OAuth

HTTP and SSE MCP servers that require OAuth can use Crush's built-in
authorization-code flow instead of a static `Authorization` header. Set
`"oauth": true` to enable it:

```json
{
  "mcp": {
    "linear": {
      "type": "http",
      "url": "https://mcp.linear.app/mcp",
      "oauth": true
    }
  }
}
```

##### Pre-registered clients

Some servers (GitHub, Slack) don't support dynamic client registration.
For those, register an OAuth app with the provider and supply the
credentials directly. All values support shell expansion:

```json
{
  "mcp": {
    "github": {
      "type": "http",
      "url": "https://api.github.com/mcp/",
      "oauth": true,
      "oauth_client_id": "Iv1.abc123def456",
      "oauth_client_secret": "$GITHUB_MCP_SECRET",
      "oauth_callback_port": 40704
    }
  }
}
```

When `oauth_client_id` is set, Crush skips dynamic client registration
and authenticates as the specified client. When omitted, Crush attempts
dynamic registration automatically (works with Linear, Notion, and other
servers that support RFC 7591).

#### Sessionless servers

Some HTTP MCP servers are sessionless — they never issue a
`Mcp-Session-Id` and reject the `subscriptions/listen` stream Crush opens
for list-changed notifications, which would otherwise break the
connection. Crush auto-detects known sessionless servers (GitHub MCP,
`api.githubcopilot.com/mcp`), so those need no extra configuration.

For other sessionless servers, mark them explicitly with
`"sessionless": true` (or `--sessionless true` in `crushrc`); set it to
`false` to force the default behavior for an auto-detected URL. The
tradeoff is that a sessionless server won't push live
tool/prompt/resource list-changed notifications.

### Hooks

Crush has preliminary support for hooks. For details, see
[the hook guide](./docs/hooks/).

### Sharing a workspace across clients

When Crush is run against a shared backend (for example two TUIs talking to
the same `crush serve`), clients are grouped into **workspaces** keyed by
their resolved `--cwd`. Two clients with the same `--cwd` join the same
underlying workspace, so they share the session list, message history,
permission queue, LSP, and MCP state.

Joining is implicit: pointing a second client at the same working directory
attaches it to the existing workspace. Each new invocation, however, starts
in its own fresh session by default. To pick up the conversation another
client already has open, use the session manager (the session picker) and
select it. Sessions surface two signals there:

- `IsBusy` is set while an agent turn is in flight for that session.
- `AttachedClients` reports how many clients are currently viewing it.

A non-zero `AttachedClients` (often combined with `IsBusy`) is the cue that a
session is "in progress" on another client and joining it will mirror that
view live.

The first client to create a workspace fixes its process-wide flags. In
particular, `--yolo` and `--debug` follow a **first-wins** rule: later
clients that arrive at the same `--cwd` with different values for those
flags do not change the running workspace. A debug log line is emitted
recording the mismatch, and the workspace keeps the flags it was created
with.

A workspace lives as long as at least one client has an SSE event stream
open against it. When the last stream disconnects, the workspace is torn
down. There is a short grace window right after `POST /v1/workspaces` so a
client that has created the workspace but not yet opened its event stream
does not get reaped before it can attach.

### Global context files

Crush automatically includes two files for cross-project instructions. Think of
these are personal additions to the system prompt.

- `~/.config/crush/CRUSH.md`: Crush-specific rules that would confuse other
  agentic coding tools. If you only use Crush, this is the only one you need to
  edit.
- `~/.config/AGENTS.md`: generic instructions that other coding tools might
  read. Avoid referring to Crush-specific features or workflows here. You
  probably only care about this if you use multiple agentic coding tools and
  want to share instructions between them.

You can customize these paths with `option global-context-path`. Repeat the
command to add multiple paths:

```bash
# Load a single markdown file.
option global-context-path "~/path/to/custom/context/file.md"

# Recursively load all Markdown files in the folder.
option global-context-path "/full/path/to/folder/of/files/"
```

### Ignoring Files

Crush respects `.gitignore` files by default, but you can also create a
`.crushignore` file to specify additional files and directories that Crush
should ignore. This is useful for excluding files that you want in version
control but don't want Crush to consider when providing context.

The `.crushignore` file uses the same syntax as `.gitignore` and can be placed
in the root of your project or in subdirectories.

### Allowing Tools

By default, Crush will ask you for permission before running tool calls. If
you'd like, you can allow tools to be executed without prompting you for
permissions. Use this with care.

```bash
permissions allow view ls grep edit mcp_context7_get-library-doc
```

The `bash` tool additionally runs a small set of read-only commands (`ls`,
`pwd`, `git status`, `git log`, …) without prompting. Crush parses the command
to decide, so it applies only when the whole command is provably inert: a
redirection, a variable assignment, a command substitution, a pipeline, or an
argument that mutates state (`git branch -D`, `git remote set-url`) all fall
back to the normal permission prompt.

### Disabling Built-In Tools

You can also deny tools, hiding then from the agent entirely:

```bash
permissions deny bash sourcegraph
```

To disable tools from MCP servers, see the [MCP config section](#mcps).

### You only live once

You can also skip all permission prompts completely by running Crush with the
`--yolo` flag. Be very, very careful with this feature.

### Disabling Skills

You can prevent Crush from using certain skills entirely. Disabled skills are
hidden from the agent, including builtin skills and skills discovered from
disk.

```bash
option disable-skill crush-config
```

### Agent Skills

Crush supports the [Agent Skills](https://agentskills.io) open standard for
extending agent capabilities with reusable skill packages. Skills are folders
containing a `SKILL.md` file with instructions that Crush can discover and
activate on demand.

The global paths we looks for skills are:

- `$CRUSH_SKILLS_DIR`
- `$XDG_CONFIG_HOME/agents/skills` or `~/.config/agents/skills/`
- `$XDG_CONFIG_HOME/crush/skills` or `~/.config/crush/skills/`
- `~/.agents/skills/`
- `~/.claude/skills/`
- On Windows, we _also_ look at
  - `%LOCALAPPDATA%\agents\skills\` or `%USERPROFILE%\AppData\Local\agents\skills\`
  - `%LOCALAPPDATA%\crush\skills\` or `%USERPROFILE%\AppData\Local\crush\skills\`
- Additional paths configured via `options.skills_paths`

On top of that, we _also_ load skills in your project from the following
relative paths:

- `.agents/skills`
- `.crush/skills`
- `.claude/skills`
- `.cursor/skills`

Or load directories of skills specifically in your config:

```bash
option skill-path "$HOME/squid-skills" "./other-skills"
```

You can get started with example skills from [anthropics/skills](https://github.com/anthropics/skills):

```bash
# Unix
mkdir -p ~/.config/crush/skills
cd ~/.config/crush/skills
git clone https://github.com/anthropics/skills.git _temp
mv _temp/skills/* . && rm -rf _temp
```

```powershell
# Windows (PowerShell)
mkdir -Force "$env:LOCALAPPDATA\crush\skills"
cd "$env:LOCALAPPDATA\crush\skills"
git clone https://github.com/anthropics/skills.git _temp
mv _temp/skills/* . ; rm -r -force _temp
```

#### User-Invocable Skills

Skills can be made invocable as commands from the commands palette
(<kbd>ctrl+p</kbd>). Add `user-invocable: true` to the skill's YAML
frontmatter:

```yaml
---
name: my-hot-skill
description: A skill that can be invoked as a command.
user-invocable: true
---
```

User-invocable skills appear in the commands palette with a `user:` or `project:` prefix:

- Skills from global directories show as `user:skill-name`
- Skills from project directories show as `project:skill-name`

When invoked, the skill's instructions are loaded into the conversation context.

To prevent the model from auto-triggering a skill (while still allowing user invocation), add `disable-model-invocation: true`:

```yaml
---
name: my-skill
description: Only invocable by users, not the model.
user-invocable: true
disable-model-invocation: true
---
```

Skills with `disable-model-invocation` won't appear in the model's available skills list but can still be invoked manually by users.

### Desktop notifications

Crush sends desktop notifications when a tool call requires permission and when
the agent finishes its turn. They're only sent when the terminal window isn't
focused _and_ your terminal supports reporting the focus state.

```bash
# Choose auto, native, osc, bell, or disabled.
option notifications disabled
```

`auto` uses native notifications locally and OSC notifications over SSH when
supported.

### Initialization

When you initialize a project, Crush analyzes your codebase and creates
a context file that helps it work more effectively in future sessions. By
default, this file is named `AGENTS.md`, but you can customize the name and
location with the `initialize-as` option:

```bash
# crushrc
option initialize-as AGENTS.md
```

This is useful if you prefer a different naming convention or want to place the
file in a specific directory (e.g., `CRUSH.md` or `docs/LLMs.md`). Crush will
fill the file with project-specific context like build commands, code patterns,
and conventions it discovered during initialization.

### Attribution Settings

By default, Crush adds attribution information to Git commits and pull requests
it creates. You can customize this behavior with `option` commands:

```bash
option attribution-trailer-style co-authored-by
option attribution-generated-with true
```

- `trailer_style`: Controls the attribution trailer added to commit messages
  (default: `assisted-by`)
  - `assisted-by`: Adds `Assisted-by: Crush:[ModelID]` as specified in [the convention](https://docs.kernel.org/process/coding-assistants.html#attribution)
  - `co-authored-by`: Adds `Co-Authored-By: Crush <crush@charm.land>`
  - `none`: No attribution trailer
- `generated_with`: When true (default), adds `💘 Generated with Crush` line to
  commit messages and PR descriptions

### Custom Providers

Crush supports custom provider configurations for both OpenAI-compatible and
Anthropic-compatible APIs.

> [!NOTE]
> The provider type describes the API format the provider speaks. Make sure
> to choose the right one to ensure the best experience!
>
> - `responses` should be used when proxying or routing requests through OpenAI.
> - `completions` should be used when using non-OpenAI providers that have OpenAI-compatible APIs.
> - `messages` should be used for Anthropic-compatible APIs.
>
> Legacy values (`openai`, `openai-compat`, `anthropic`) are still accepted and
> migrated automatically.

#### OpenAI-Compatible APIs

Here’s an example configuration for Deepseek, which uses an OpenAI-compatible
API. Don't forget to set `DEEPSEEK_API_KEY` in your environment.

```bash
provider add deepseek --type completions \
  --base-url "https://api.deepseek.com/v1" \
  --api-key "$DEEPSEEK_API_KEY"

model add deepseek/deepseek-chat \
  --name "Deepseek V3" \
  --context-window 64000 \
  --default-max-tokens 5000 \
  --price-input 0.27 \
  --price-output 1.1 \
  --price-cache-create 1.1 \
  --price-cache-hit 0.07
```

#### Anthropic-Compatible APIs

Custom Anthropic-compatible providers follow this format:

```bash
provider add custom-anthropic \
  --type messages \
  --base-url "https://api.anthropic.com/v1" \
  --api-key "$ANTHROPIC_API_KEY" \
  --extra-header anthropic-version 2023-06-01

model add custom-anthropic/claude-sonnet-4-20250514 \
  --name "Claude Sonnet 4" \
  --context-window 200000 \
  --default-max-tokens 50000 \
  --thinking toggleable \
  --supports-images true \
  --price-input 3 \
  --price-output 15 \
  --price-cache-create 3.75 \
  --price-cache-hit 0.3
```

### Amazon Bedrock

Crush currently supports running Anthropic models through Bedrock, with caching disabled.

A Bedrock provider appears once Crush can find AWS credentials. You can
authenticate in one of two ways:

**API key.** Set `AWS_BEARER_TOKEN_BEDROCK` to a Bedrock API key. This is the
simplest option and never expires mid-session.

**AWS credential chain (SSO, profiles, access keys).** Configure AWS the usual
way with `aws configure` or `aws configure sso`. Crush picks up whatever the
AWS SDK credential chain resolves, including `AWS_PROFILE`, `AWS_ACCESS_KEY_ID`
/ `AWS_SECRET_ACCESS_KEY`, or an SSO session. To select a specific profile,
set `AWS_PROFILE` in your shell (`AWS_PROFILE=myprofile crush`) or in the
top-level [`env`](#environment-variables) config.

If you authenticate via AWS SSO, your session expires periodically. Set
`aws_auth_refresh` to a command that refreshes it. When Bedrock returns a
credential error, Crush runs the command, then retries the request in place
(no duplicate messages, no manual restart):

```json
{
  "$schema": "https://charm.land/crush.json",
  "env": {
    "AWS_PROFILE": "my-sso-profile"
  },
  "providers": {
    "bedrock": {
      "aws_auth_refresh": "aws sso login --profile my-sso-profile"
    },
    "bedrock-europe": {
      "aws_auth_refresh": "aws sso login --profile my-eu-sso-profile"
    }
  }
}
```

- `aws_auth_refresh` — shell command run when AWS credentials expire (e.g. `aws sso login`)

### Vertex AI Platform

Vertex AI will appear in the list of available providers when `VERTEXAI_PROJECT` and `VERTEXAI_LOCATION` are set. You will also need to be authenticated:

```bash
$ gcloud auth application-default login
```

To add specific models to the configuration, configure as such:

```bash
# crushrc — authentication still comes from gcloud and the VERTEXAI_* env vars.
provider add vertexai

model add vertexai/claude-sonnet-4@20250514 \
  --name "VertexAI Sonnet 4" \
  --context-window 200000 \
  --default-max-tokens 50000 \
  --thinking toggleable \
  --supports-images true \
  --price-input 3 \
  --price-output 15 \
  --price-cache-create 3.75 \
  --price-cache-hit 0.3
```

### Local Models

Crush can auto-discovers models from local providers. Add a custom provider
with `type` set to `llamacpp`, `omlx`, `lmstudio`, `litellm`, or `ollama`
and leave out the models list. Crush will populate the model list
automatically.

```bash
# Piece of cake.
provider add ollama \
  --name Ollama \
  --type ollama \
  --base-url "http://localhost:11434/v1/"
```

For llama.cpp (`llama-server`), point at the server's base URL:

```bash
provider add llamacpp \
  --name "llama.cpp" \
  --type llamacpp \
  --base-url "http://localhost:2222"
```

#### Manual Model Configuration

You can still list models explicitly. User-defined models always take
precedence over discovered ones, and any fields you set won't be overwritten
by auto-discovery. Auto discovery will run if the model list is empty for any
`openai-compat` provider or if you pass `"discover_models": true` it will merge
the found models with your hand configured ones.

```bash
# crushrc
provider add ollama \
  --name Ollama \
  --type ollama \
  --base-url "http://localhost:11434/v1/" \
  --discover-models true

model add ollama/qwen3:30b \
  --name "Qwen 3 30B" \
  --context-window 256000 \
  --default-max-tokens 20000
```

The `--discover-models true` flag merges discovered models with the one above;
your explicit model fields win on conflicts.

## Logging

Sometimes you need to look at logs. Luckily, Crush logs all sorts of
stuff. Logs are stored in `./.crush/logs/crush.log` relative to the project.

The CLI also contains some helper commands to make perusing recent logs easier:

```bash
# Print the last 1000 lines
crush logs

# Print the last 500 lines
crush logs --tail 500

# Follow logs in real time
crush logs --follow
```

Want more logging? Run `crush` with the `--debug` flag, or enable it in your
`crushrc`:

```bash
# crushrc
option debug true
option debug-lsp true
```

## Provider Auto-Updates

By default, Crush automatically checks for the latest and greatest list of
providers and models from [Catwalk](https://github.com/charmbracelet/catwalk),
the open source Crush provider database. This means that when new providers and
models are available, or when model metadata changes, Crush automatically
updates your local configuration.

### Custom provider catalog

You can also override [Catwalk](https://github.com/charmbracelet/catwalk) default URL (for testing, using a fork).

You can do so by setting the `CATWALK_URL` environment variable. (e.g. `export CATWALK_URL=http://localhost:8000`)

### Disabling automatic provider updates

For those with restricted internet access, or those who prefer to work in
air-gapped environments, this might not be what you want, and this feature can
be disabled.

To disable automatic provider updates in your `crushrc`:

```bash
option provider-auto-update false
```

Or set the `CRUSH_DISABLE_PROVIDER_AUTO_UPDATE` environment variable:

```bash
export CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1
```

### Manually updating providers

Manually updating providers is possible with the `crush update-providers`
command:

```bash
# Update providers remotely from Catwalk.
crush update-providers

# Update providers from a custom Catwalk base URL.
crush update-providers https://example.com/

# Update providers from a local file.
crush update-providers /path/to/local-providers.json

# Reset providers to the embedded version, embedded at crush at build time.
crush update-providers embedded

# For more info:
crush update-providers --help
```

## Metrics

Crush records pseudonymous usage metrics (tied to a device-specific hash),
which maintainers rely on to inform development and support priorities. The
metrics include solely usage metadata; prompts and responses are NEVER
collected.

Details on exactly what’s collected are in the source code ([here](https://github.com/charmbracelet/crush/tree/main/internal/event)
and [here](https://github.com/charmbracelet/crush/blob/main/internal/llm/agent/event.go)).

Also, when you use Crush with [Hyper][hyper], your Crush usage metrics may
be associated with your Hyper account to help us improve the service and
diagnose issues. For more information, see [the Hyper Privacy Policy](https://hyper.charm.land/legal/privacy-policy).

Either way, you can opt out of metrics collection at any time by setting
the following  in your environment:

```bash
export CRUSH_DISABLE_METRICS=1
```

Crush also respects the [`DO_NOT_TRACK`](https://donottrack.sh/) convention
which can be enabled via `export DO_NOT_TRACK=1`.

## Q&A

### Why is clipboard copy and paste not working?

Installing an extra tool might be needed on Unix-like environments.

| Environment         | Tool                     |
| ------------------- | ------------------------ |
| Windows             | Native support           |
| macOS               | Native support           |
| Linux/BSD + Wayland | `wl-copy` and `wl-paste` |
| Linux/BSD + X11     | `xclip` or `xsel`        |

## Contributing

See the [contributing guide](https://github.com/charmbracelet/crush?tab=contributing-ov-file#contributing).

## Whatcha think?

We’d love to hear your thoughts on this project. Need help? We gotchu. You can find us on:

- [Twitter](https://twitter.com/charmcli)
- [Slack][slack]
- [Discord][discord]
- [The Fediverse](https://mastodon.social/@charmcli)
- [Bluesky](https://bsky.app/profile/charm.land)

[slack]: https://charm.land/slack
[discord]: https://charm.land/discord

## License

[FSL-1.1-MIT](https://github.com/charmbracelet/crush/raw/main/LICENSE.md)

---

Part of [Charm](https://charm.land).

<a href="https://charm.land/"><img alt="The Charm logo" width="400" src="https://stuff.charm.sh/charm-banner-softy.jpg" /></a>

<!--prettier-ignore-->
Charm热爱开源 • Charm loves open source

</details>
