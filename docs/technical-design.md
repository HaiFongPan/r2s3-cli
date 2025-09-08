# R2S3-CLI 技术设计文档

## 项目概述

R2S3-CLI 是一个用于管理 Cloudflare R2 存储桶的命令行工具，主要用于图片的上传、删除和预览功能。项目使用 Go 语言开发，采用现代化的 CLI 工具设计模式。

## 技术架构

### 核心技术栈

- **Go 1.25+**: 主要开发语言
- **Viper**: 配置管理（支持 TOML、环境变量、命令行参数）
- **Cobra**: CLI 框架，用于构建命令和子命令
- **AWS SDK for Go v2**: 与 Cloudflare R2 (S3兼容) 交互
- **TOML**: 配置文件格式

### 项目结构

```
r2s3-cli/
├── cmd/                    # 命令定义
│   ├── root.go            # 根命令和全局配置
│   ├── upload.go          # 上传命令
│   ├── delete.go          # 删除命令
│   ├── list.go            # 列表命令
│   └── preview.go         # 预览命令
├── internal/              # 内部包
│   ├── config/           # 配置管理
│   │   ├── config.go     # 配置结构和加载
│   │   └── validate.go   # 配置验证
│   ├── r2/               # R2 客户端封装
│   │   ├── client.go     # R2 客户端
│   │   └── operations.go # 上传、删除等操作
│   └── utils/            # 工具函数
│       ├── file.go       # 文件操作
│       └── logger.go     # 日志工具
├── docs/                 # 文档
│   ├── technical-design.md
│   └── user-guide.md
├── examples/             # 示例配置
│   └── config.toml
├── main.go              # 程序入口
├── go.mod
└── go.sum
```

## 配置管理设计

### 配置优先级

1. 命令行参数 (最高优先级)
2. 环境变量
3. 配置文件 (TOML)
4. 默认值 (最低优先级)

### 配置结构

```go
type Config struct {
    R2 R2Config `mapstructure:"r2"`
    Log LogConfig `mapstructure:"log"`
    General GeneralConfig `mapstructure:"general"`
}

type R2Config struct {
    AccountID       string `mapstructure:"account_id"`
    AccessKeyID     string `mapstructure:"access_key_id"`
    AccessKeySecret string `mapstructure:"access_key_secret"`
    BucketName      string `mapstructure:"bucket_name"`
    Endpoint        string `mapstructure:"endpoint"`
    Region          string `mapstructure:"region"`
}

type LogConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
}

type GeneralConfig struct {
    DefaultTimeout int    `mapstructure:"default_timeout"`
    MaxRetries     int    `mapstructure:"max_retries"`
    ConfigPath     string `mapstructure:"config_path"`
}
```

### 配置文件示例 (TOML)

```toml
[r2]
account_id = "your-account-id"
access_key_id = "your-access-key"
access_key_secret = "your-secret-key"
bucket_name = "your-bucket-name"
endpoint = "auto"
region = "auto"

[log]
level = "info"
format = "text"

[general]
default_timeout = 30
max_retries = 3
```

### 环境变量

- `R2CLI_ACCOUNT_ID`
- `R2CLI_ACCESS_KEY_ID`
- `R2CLI_ACCESS_KEY_SECRET`
- `R2CLI_BUCKET_NAME`
- `R2CLI_LOG_LEVEL`
- `R2CLI_CONFIG_PATH`

## CLI 命令设计

### 根命令

```bash
r2s3-cli [global options] command [command options] [arguments...]
```

### 全局参数

- `--config, -c`: 指定配置文件路径
- `--verbose, -v`: 启用详细输出
- `--quiet, -q`: 静默模式
- `--help, -h`: 显示帮助信息

### 子命令

#### 1. upload - 上传文件

```bash
r2s3-cli upload [options] <file-path> [remote-path]
```

**参数:**
- `file-path`: 本地文件路径 (必需)
- `remote-path`: 远程存储路径 (可选，默认使用文件名)

**选项:**
- `--bucket, -b`: 指定存储桶名称
- `--public`: 设置文件为公开访问
- `--content-type, -t`: 指定内容类型
- `--overwrite`: 覆盖已存在的文件

**示例:**
```bash
r2s3-cli upload image.jpg
r2s3-cli upload image.jpg photos/2023/image.jpg --public
r2s3-cli upload *.jpg photos/batch/ --overwrite
```

#### 2. delete - 删除文件

```bash
r2s3-cli delete [options] <remote-path>
```

**参数:**
- `remote-path`: 远程文件路径 (必需)

**选项:**
- `--bucket, -b`: 指定存储桶名称
- `--force, -f`: 强制删除，不显示确认提示
- `--recursive, -r`: 递归删除目录

**示例:**
```bash
r2s3-cli delete photos/image.jpg
r2s3-cli delete photos/ --recursive --force
```

#### 3. list - 列出文件

```bash
r2s3-cli list [options] [prefix]
```

**参数:**
- `prefix`: 文件前缀过滤 (可选)

**选项:**
- `--bucket, -b`: 指定存储桶名称
- `--limit, -l`: 限制返回数量
- `--format, -f`: 输出格式 (table|json|yaml)
- `--size`: 显示文件大小
- `--date`: 显示修改时间

**示例:**
```bash
r2s3-cli list
r2s3-cli list photos/ --format json
r2s3-cli list --limit 10 --size --date
```

#### 4. preview - 预览/下载文件

```bash
r2s3-cli preview [options] <remote-path>
```

**参数:**
- `remote-path`: 远程文件路径 (必需)

**选项:**
- `--bucket, -b`: 指定存储桶名称
- `--download, -d`: 下载到本地
- `--output, -o`: 指定输出文件路径
- `--url`: 生成预签名 URL

**示例:**
```bash
r2s3-cli preview photos/image.jpg --url
r2s3-cli preview photos/image.jpg --download --output ./downloaded.jpg
```

## 核心功能模块

### 1. R2 客户端封装

```go
type Client struct {
    s3Client *s3.Client
    config   *config.R2Config
}

func NewClient(cfg *config.R2Config) (*Client, error)
func (c *Client) Upload(ctx context.Context, req *UploadRequest) error
func (c *Client) Delete(ctx context.Context, req *DeleteRequest) error
func (c *Client) List(ctx context.Context, req *ListRequest) (*ListResponse, error)
func (c *Client) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error)
```

### 2. 错误处理

- 使用 Go 1.20+ 的 error wrapping
- 定义自定义错误类型
- 提供用户友好的错误信息
- 支持调试模式显示详细错误堆栈

### 3. 日志系统

- 支持多级日志 (DEBUG, INFO, WARN, ERROR)
- 可配置日志格式 (JSON, Text)
- 支持日志文件输出
- 结构化日志记录

## 安全考虑

### 1. 凭证管理

- **绝不在代码中硬编码凭证**
- 支持多种凭证来源：
  - 配置文件 (加密存储)
  - 环境变量
  - AWS 凭证链
- 配置文件权限检查 (600)

### 2. 输入验证

- 文件路径验证
- 存储桶名称格式检查
- 文件大小限制
- 文件类型白名单

### 3. 网络安全

- 强制使用 HTTPS
- 请求超时设置
- 重试机制和退避策略

## 性能优化

### 1. 并发上传

- 支持分片上传大文件
- 并发控制（限制并发数）
- 进度显示

### 2. 缓存机制

- 文件元数据缓存
- 预签名 URL 缓存

### 3. 资源管理

- 连接池复用
- 内存使用优化
- 临时文件清理

## 测试策略

### 1. 单元测试

- 配置管理模块测试
- R2 客户端模拟测试
- 工具函数测试

### 2. 集成测试

- 实际 R2 服务集成测试
- 配置文件解析测试
- CLI 命令端到端测试

### 3. 性能测试

- 大文件上传性能测试
- 并发操作压力测试
- 内存泄漏检测

## 部署和分发

### 1. 编译

- 支持多平台编译 (Linux, macOS, Windows)
- 静态链接，无外部依赖
- 版本信息内嵌

### 2. 分发方式

- GitHub Releases
- Go 模块
- 包管理器 (Homebrew, apt, yum)
- Docker 镜像

## 开发计划

### Phase 1: 基础架构
- [ ] 项目结构搭建
- [ ] 配置管理实现
- [ ] CLI 框架搭建
- [ ] R2 客户端封装

### Phase 2: 核心功能
- [ ] 上传功能实现
- [ ] 删除功能实现
- [ ] 列表功能实现
- [ ] 预览功能实现

### Phase 3: 增强功能
- [ ] 批量操作
- [ ] 进度显示
- [ ] 错误重试
- [ ] 日志和调试

### Phase 4: 优化和发布
- [ ] 性能优化
- [ ] 测试完善
- [ ] 文档编写
- [ ] 发布准备

## 扩展性考虑

### 1. 插件系统

- 支持自定义插件
- 钩子机制
- 配置扩展

### 2. 多云支持

- 抽象存储接口
- 支持 AWS S3, Google Cloud Storage 等
- 统一配置格式

### 3. 高级功能

- 文件同步
- 备份策略
- 访问控制
- 审计日志

---

## 附录

### A. 依赖包列表

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/aws/aws-sdk-go-v2 v1.24.0
    github.com/aws/aws-sdk-go-v2/config v1.26.0
    github.com/aws/aws-sdk-go-v2/service/s3 v1.47.0
    github.com/BurntSushi/toml v1.3.0
    github.com/sirupsen/logrus v1.9.0
    github.com/stretchr/testify v1.8.0
)
```

### B. 配置文件模板

详见 `examples/config.toml`

### C. 环境变量列表

完整的环境变量配置参考见用户指南。