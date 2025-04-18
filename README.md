# 🧙‍♂️ mcpenetes

![mcpenetes](https://img.shields.io/badge/mcpenetes-MCP%20Configuration%20Manager-blue)
![License](https://img.shields.io/badge/license-MIT-green)

> *"One CLI to rule them all, one CLI to find them, one CLI to bring them all, and in the configurations bind them."*

## 🌟 What is mcpenetes?

**mcpenetes** is a magical CLI tool that helps you manage multiple Model Context Protocol (MCP) server configurations with ease! If you're tired of manually editing config files for different MCP-compatible clients whenever you want to switch servers, mcpenetes is here to save your day.

Think of mcpenetes as your friendly neighborhood wizard who can:

- 🔍 Search for available MCP servers from configured registries
- 🔄 Switch between different MCP server configurations
- 🧠 Apply configurations across all your MCP clients automatically
- 💾 Backup your configurations before making any changes
- 🛡️ Restore configurations if something goes wrong

## 🚀 Installation

### From Source

```bash
git clone https://github.com/tuannvm/mcpenetes.git
cd mcpenetes
make build
# The binary will be available at ./bin/mcpenetes
```

### Using Go

```bash
go install github.com/tuannvm/mcpenetes@latest
```

## 🏄‍♂️ Quick Start

1. **Search for available MCP servers**:

```bash
mcpenetes search
```

2. **Apply selected configuration** to all your clients:

```bash
mcpenetes apply
```

That's it! Your MCP configurations are now synced across all clients. Magic! ✨

## 📚 Usage Guide

### 🛠️ Available Commands

```
search         Interactive fuzzy search for MCP versions and apply them
apply          Applies MCP configuration to all clients
load           Load MCP server configuration from clipboard
restore        Restores client configurations from the latest backups
```

### 📋 Searching for MCP Servers

The `search` command lets you interactively find and select MCP servers from configured registries. It will present you with a list of available servers that you can select from.

```bash
mcpenetes search
```

You can also directly specify a server ID:

```bash
mcpenetes search claude-3-opus-0403
```

By default, search results are cached to improve performance. Use the `--refresh` flag to force a refresh:

```bash
mcpenetes search --refresh
```

### 📥 Loading Configuration from Clipboard

If you've copied an MCP configuration to your clipboard, you can load it directly:

```bash
mcpenetes load
```

### 🗑️ Removing Resources

To remove a registry:

```bash
mcpenetes remove registry my-registry
```

### ⏪ Restoring Configurations

If something goes wrong, you can restore your clients' configurations from backups:

```bash
mcpenetes restore
```

## 🧩 Supported Clients

mcpenetes automatically detects and configures the following MCP-compatible clients:

- Claude Desktop
- Windsurf
- Cursor
- Visual Studio Code extensions

## 📁 Configuration Files

mcpenetes uses the following configuration files:

- `~/.config/mcpenetes/config.yaml`: Stores global configuration, including registered registries and selected MCP servers
- `~/.config/mcpenetes/mcp.json`: Stores the MCP server configurations
- `~/.config/mcpenetes/cache/`: Caches registry responses for faster access

## 🤝 Contributing

Contributions are welcome! Feel free to:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

Licensed under the MIT License. See the LICENSE file for details.

## 🌐 Related Projects

- [mcp-trino](https://github.com/tuannvm/mcp-trino): Trino MCP server implementation

---

Made with ❤️ by humans (and occasionally with the help of AI)
