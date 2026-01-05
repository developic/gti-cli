
# <p align="center">GTI</p>

<p align="center">
  <a href="https://github.com/developic/gti-cli/stargazers">
    <img src="https://img.shields.io/github/stars/developic/gti-cli?style=flat&cacheSeconds=60" alt="GitHub stars">
  </a>
  <a href="https://github.com/developic/gti-cli/issues">
    <img src="https://img.shields.io/github/issues/developic/gti-cli.svg?color=orange" alt="GitHub issues">
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT">
  </a>
  <a href="https://golang.org">
    <img src="https://img.shields.io/badge/Go-1.19+-blue.svg" alt="Go Version">
  </a>
</p>

GTI (Graphical Typing Tool) is a fast, lightweight and terminal based typing speed and practice application.

> [!IMPORTANT] 
> perfect for developers who want a minimal, distraction-free CLI experience

---

## Installation

### Build from Source
```bash
git clone https://github.com/developic/gti-cli
cd gti-cli
go build -o gti main.go
```

### Linux
```bash
sudo curl -L https://github.com/developic/gti-cli/releases/download/v1.0.0/gti-linux -o /usr/local/bin/gti && sudo chmod +x /usr/local/bin/gti
sudo curl -o /usr/share/man/man1/gti.1.gz -L https://github.com/developic/gti-cli/releases/download/v1.0.0/gti.1.gz
```

### macOS
```bash
mkdir -p /usr/local/bin /usr/local/share/man/man1
sudo curl -L https://github.com/developic/gti-cli/releases/download/v1.0.0/gti-mac -o /usr/local/bin/gti && sudo chmod +x /usr/local/bin/gti
sudo curl -o /usr/local/share/man/man1/gti.1.gz -L https://github.com/developic/gti-cli/releases/download/v1.0.0/gti.1.gz
```

### Windows
```powershell
if (-Not (Test-Path "C:\Tools")) { New-Item -ItemType Directory -Path "C:\Tools" }; 
curl -L -o "$env:USERPROFILE\gti.exe" https://github.com/developic/gti-cli/releases/download/v1.0.0/gti.exe; 
Move-Item -Force "$env:USERPROFILE\gti.exe" "C:\Tools\gti.exe"; 
if ($env:Path -notlike "*C:\Tools*") { [Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Tools", [EnvironmentVariableTarget]::User) }
```

### Uninstall

#### Linux
```bash
sudo rm /usr/local/bin/gti /usr/share/man/man1/gti.1.gz
```

#### Windows
```powershell
Remove-Item "C:\Tools\gti.exe" -Force
```

---

## Features

- **Practice Modes**: Default practice with configurable chunks and groups
- **Timed Tests**: Set custom time limits for focused practice sessions
- **Custom Text**: Practice with your own text files
- **Random Quotes**: Type inspirational and famous quotes
- **Progressive Challenges**: Level-based challenges with increasing difficulty
- **Statistics Tracking**: Comprehensive typing statistics and progress tracking
- **Multi-language Support**: Practice in 25+ languages including English, Spanish, French, German, Japanese, and more
- **Theme System**: 25+ color themes for terminal customization
- **Configuration Management**: Persistent settings and preferences

---

## Usage

### Quick Start
```bash
# Start practice mode (2 chunks)
gti

# Practice with 10 chunks per group
gti -n 10

# Start 30-second timed test
gti -t 30

# Practice with custom text file
gti -c text.txt
```

### Commands

| Command | Description |
|---------|-------------|
| `gti` | Start practice mode |
| `gti quote` | Start with random quotes |
| `gti challenge` | Progressive challenge with levels |
| `gti statistics` | View detailed typing statistics |
| `gti theme` | Manage color themes |
| `gti config` | View and manage configuration |
| `gti version` | Display version information |

### Options

| Option | Description |
|--------|-------------|
| `-n <count>` | Number of chunks per group (default: 2) |
| `-g <count>` | Number of groups (default: 1) |
| `-c, --custom <file>` | Start with custom text file |
| `--start <num>` | Start from paragraph number (for custom mode) |
| `-t, --timed <time>` | Start timed mode (e.g., 30, 10s, 5m) |
| `-l, --language <lang>` | Language for word generation |
| `-s, --shortcuts` | Show shortcuts and exit |

### Examples
```bash
# 30-second timed test
gti -t 30

# Practice with 5 chunks per group, 3 groups total
gti -n 5 -g 3

# Custom text starting from paragraph 5
gti -c document.txt --start 5

# Practice in Spanish
gti -l spanish

# Show keyboard shortcuts
gti -s
```

---

## Configuration

GTI stores configuration in your system's config directory. Use the `config` command to view and manage settings:

```bash
gti config --show     # View current configuration
gti config --reset    # Reset to defaults
```

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+C` | Force quit application |
| `Ctrl+Q` | Quit with confirmation |
| `Tab/Enter` | Submit completed text |
| `Ctrl+R` | Restart current session |
| `Esc` | Close overlays/Cancel operations |

---

## Supported Languages

GTI supports **25+ languages**:  
English, Spanish, French, German, Japanese, Russian, Italian, Portuguese, Chinese, Arabic, Hindi, Korean, Dutch, Swedish, Czech, Danish, Finnish, Greek, Hebrew, Hungarian, Norwegian, Polish, Thai, Turkish.

---

## Contributing

We welcome contributions! Here's how you can help:

- **Report Bugs**: Open an issue with detailed information
- **Suggest Features**: Share your ideas for new features
- **Code Contributions**: Submit pull requests with improvements
- **Documentation**: Help improve documentation

---

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

