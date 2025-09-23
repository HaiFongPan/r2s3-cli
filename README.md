# R2S3-CLI

A simple command-line tool for managing Cloudflare R2 storage buckets, Written with Claude Code and Zed.

## Features

- Upload, download, list, and delete files in Cloudflare R2
- Interactive TUI file browser with progress bars
- Support for custom domains and URL generation
- Basic image compression before upload
- Configuration via TOML files or environment variables

## Installation

### From Source

```bash
git clone https://github.com/HaiFongPan/r2s3-cli.git
cd r2s3-cli
go build -o r2s3-cli
```

### Using Go Install

```bash
go install github.com/HaiFongPan/r2s3-cli@latest
```

## Quick Start

1. **Create config file:**

```bash
mkdir -p ~/.r2s3-cli
cp examples/config.toml ~/.r2s3-cli/config.toml
```

2. **Edit the config with your R2 credentials:**

```toml
[r2]
account_id = "your-cloudflare-account-id"
access_key_id = "your-r2-access-key-id"
access_key_secret = "your-r2-secret-access-key"
bucket_name = "your-bucket-name"
```

3. **Start using the tool:**

```bash
r2s3-cli
```

![Preview](https://images.bugnone.dev/%2Fr2s3cli%2Fmain.png)

```bash
r2s3-cli --Help

R2S3-CLI is a command line tool for managing files in Cloudflare R2 storage.
It supports uploading, downloading, deleting, and listing files with flexible
configuration management using TOML files, environment variables, and CLI flags.

Example usage:
  r2s3-cli # Interactive file browser
  r2s3-cli upload image.jpg
  r2s3-cli list photos/
  r2s3-cli delete old-file.jpg

Usage:
  r2s3-cli [flags]
  r2s3-cli [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  delete      Delete a file from R2 storage
  help        Help about any command
  list        List files in the R2 bucket
  upload      Upload a file or folder to R2 storage

Flags:
  -c, --config string   config file (default is ~/.r2s3-cli/config.toml)
  -h, --help            help for r2s3-cli
  -q, --quiet           enable quiet mode
  -v, --verbose         enable verbose output

Use "r2s3-cli [command] --help" for more information about a command.
```

## Configuration

### Getting R2 Credentials

1. Go to Cloudflare Dashboard → R2 Object Storage
2. Click "Manage R2 API tokens"
3. Create a new token
4. Copy your Account ID, Access Key ID, and Secret Access Key

> To change the bucket in TUI mode, your Account API Token needs the 'Admin Read & Write' permission. Otherwise, you can't proceed.

## Commands

### Upload

```bash
r2s3-cli upload file.jpg                    # Upload file
r2s3-cli upload file.jpg --compress normal  # Upload with compression
```

### List

```bash
r2s3-cli list                     # List all files
r2s3-cli list photos/             # List with prefix
```

### Delete

```bash
r2s3-cli delete file.jpg                    # Delete with confirmation
r2s3-cli delete file.jpg --force            # Delete without confirmation
r2s3-cli delete photos/ --recursive         # Delete with prefix
```

> Operations like prefix search, upload, and delete are also available in TUI mode.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

MIT License

Copyright (c) 2025 HaiFongPan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
