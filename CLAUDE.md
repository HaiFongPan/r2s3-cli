# Claude Code Development Rules

## Development Rules

1. **每次修改结束，必须确保单测通过**
   - 运行 `go test ./...` 确保所有测试通过
   - 如果有测试失败，必须修复后才能继续
   - 新功能必须包含相应的单元测试

2. **每次修改结束，必须成功构建应用**
   - 运行 `go build .` 确保编译无错误
   - 检查所有依赖项和导入是否正确
   - 确保代码遵循 Go 语言规范

## Testing Commands

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/tui

# 运行测试并显示详细输出
go test -v ./...

# 运行测试并显示覆盖率
go test -cover ./...
```

## Build Commands

```bash
# 基本构建
go build .

# 构建到指定目录
go build -o bin/r2s3-cli .

# 交叉编译（示例）
GOOS=linux GOARCH=amd64 go build -o bin/r2s3-cli-linux .
```

## Code Quality Guidelines

- 遵循 Go 语言官方代码规范
- 使用 `go fmt` 格式化代码
- 使用 `go vet` 检查代码问题
- 保持函数简洁，单一职责
- 添加必要的注释和文档

## Git Workflow

- 每次修改后先测试和构建
- 提交前确保所有检查通过
- 提交信息要清晰描述修改内容
- 大的功能改动要分步提交

## Project Structure

```
.
├── cmd/                 # 命令行接口
├── internal/           # 内部包
│   ├── config/        # 配置管理
│   ├── r2/            # R2 客户端
│   ├── tui/           # 终端用户界面
│   └── utils/         # 工具函数
├── examples/          # 示例配置文件
├── main.go           # 主入口
├── go.mod            # Go 模块定义
└── README.md         # 项目说明
```