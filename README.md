# R2S3-CLI

A simple command-line tool for managing Cloudflare R2 storage buckets.

## Features

- Upload, download, list, and delete files in Cloudflare R2
- Interactive TUI file browser with progress bars
- Support for custom domains and URL generation
- Basic image compression before upload
- Configuration via TOML files or environment variables

## Installation

### From Source

```bash
git clone https://github.com/yourusername/r2s3-cli.git
cd r2s3-cli
go build -o r2s3-cli
```

### Using Go Install

```bash
go install github.com/yourusername/r2s3-cli@latest
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
r2s3-cli list                    # List files
r2s3-cli list --interactive      # Interactive browser
r2s3-cli upload file.jpg         # Upload file
r2s3-cli download file.jpg       # Download file
r2s3-cli delete file.jpg         # Delete file
```

## Configuration

### Environment Variables

```bash
export R2CLI_ACCOUNT_ID="your-cloudflare-account-id"
export R2CLI_ACCESS_KEY_ID="your-r2-access-key-id"  
export R2CLI_ACCESS_KEY_SECRET="your-r2-secret-access-key"
export R2CLI_BUCKET_NAME="your-bucket-name"
```

### Getting R2 Credentials

1. Go to Cloudflare Dashboard → R2 Object Storage
2. Click "Manage R2 API tokens"
3. Create a new token
4. Copy your Account ID, Access Key ID, and Secret Access Key

## Commands

### Upload
```bash
r2s3-cli upload file.jpg                    # Upload file
r2s3-cli upload file.jpg --public           # Upload with public access
r2s3-cli upload file.jpg --compress normal  # Upload with compression
```

### List
```bash
r2s3-cli list                     # List all files  
r2s3-cli list photos/             # List with prefix
r2s3-cli list --interactive       # Interactive browser
r2s3-cli list --format json       # JSON output
```

### Download
```bash
r2s3-cli download file.jpg                    # Download to current dir
r2s3-cli download file.jpg --output /tmp/     # Download to specific path
```

### Delete
```bash
r2s3-cli delete file.jpg                    # Delete with confirmation
r2s3-cli delete file.jpg --force            # Delete without confirmation
r2s3-cli delete photos/ --recursive         # Delete with prefix
```

## Interactive Browser Controls

- `↑/↓` or `k/j`: Navigate
- `Enter`: Download file
- `d`: Delete file  
- `p`: Show file info
- `r`: Refresh
- `?`: Help
- `q/Esc`: Quit

## Image Compression Levels

- `high`: 95% JPEG quality
- `normal`: 75% JPEG quality  
- `low`: 60% JPEG quality

## License

MIT License