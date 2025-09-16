package image

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BourgeoisBear/rasterm"
)

// 渲染配置
const (
	DefaultMaxWidth  = 120
	DefaultMaxHeight = 48
	MaxImageWidth    = 8192
	MaxImageHeight   = 4320
)

// TerminalType 终端类型
type TerminalType string

const (
	TerminalKitty   TerminalType = "kitty"
	TerminalITerm2  TerminalType = "iterm2"
	TerminalWezTerm TerminalType = "wezterm"
	TerminalGhostty TerminalType = "ghostty"
	TerminalGeneric TerminalType = "generic"
)

// GraphicsProtocol 图形协议
type GraphicsProtocol string

const (
	ProtocolKitty GraphicsProtocol = "kitty"
	ProtocolITerm GraphicsProtocol = "iterm2"
	ProtocolSixel GraphicsProtocol = "sixel"
	ProtocolNone  GraphicsProtocol = "none"
)

// RendererCapabilities 渲染器能力
type RendererCapabilities struct {
	MaxWidth         int
	MaxHeight        int
	SupportedFormats []ImageFormat
	HasAlpha         bool
	HasAnimation     bool
}

// ImageRenderer 封装 rasterm 库的图片渲染功能
type ImageRenderer struct {
	TerminalType TerminalType
	Protocol     GraphicsProtocol
	MaxWidth     int
	MaxHeight    int
	Supported    bool

	// 渲染选项
	Quality    int
	PreserveAR bool // 保持宽高比
	AutoResize bool

	// 文本模式（ANSI 彩色块）渲染配置
	TextMode bool
	// 文本模式下用于渲染的单元格尺寸
	CellCols int
	CellRows int
}

// ImageRendererInterface 定义渲染器接口
type ImageRendererInterface interface {
	DetectTerminal() (TerminalType, GraphicsProtocol, error)
	RenderImage(imagePath string, maxWidth, maxHeight int) (string, error)
	RenderImageAtCells(imagePath string, cols, rows, startCol, startRow int) (string, error)
	IsSupported() bool
	GetCapabilities() RendererCapabilities
	SetMaxSize(width, height int)
	SetQuality(quality int)
	ClearScreen() error
	SetTextMode(enabled bool)
	SetCellSize(cols, rows int)
}

// NewImageRenderer 创建新的图片渲染器
func NewImageRenderer() *ImageRenderer {
	renderer := &ImageRenderer{
		MaxWidth:   DefaultMaxWidth,
		MaxHeight:  DefaultMaxHeight,
		Quality:    85, // 默认质量
		PreserveAR: true,
		AutoResize: true,
		TextMode:   false,
	}

	// 自动检测终端类型
	termType, protocol, err := renderer.DetectTerminal()
	renderer.TerminalType = termType
	renderer.Protocol = protocol
	renderer.Supported = (err == nil && protocol != ProtocolNone)

	return renderer
}

// DetectTerminal 检测终端类型并选择图形协议
func (r *ImageRenderer) DetectTerminal() (TerminalType, GraphicsProtocol, error) {
	// 检查环境变量来判断终端类型
	term := strings.ToLower(os.Getenv("TERM"))
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	kittyWindow := os.Getenv("KITTY_WINDOW_ID")
	ghosttyPath := os.Getenv("GHOSTTY")

	// Kitty 终端检测
	if kittyWindow != "" || strings.Contains(term, "kitty") {
		return TerminalKitty, ProtocolKitty, nil
	}

	// Ghostty 终端检测 - 使用 Kitty 协议
	if ghosttyPath != "" || termProgram == "ghostty" || strings.Contains(term, "ghostty") {
		return TerminalGhostty, ProtocolKitty, nil
	}

	// iTerm2 检测
	if termProgram == "iterm.app" {
		return TerminalITerm2, ProtocolITerm, nil
	}

	// WezTerm 检测
	if termProgram == "wezterm" {
		return TerminalWezTerm, ProtocolITerm, nil // WezTerm 支持 iTerm2 协议
	}

	// 检查是否支持 Sixel
	if r.supportsSixel() {
		return TerminalGeneric, ProtocolSixel, nil
	}

	// 默认不支持图形
	return TerminalGeneric, ProtocolNone, nil
}

// supportsSixel 检测终端是否支持 Sixel
func (r *ImageRenderer) supportsSixel() bool {
	// 简单检测：查看 TERM 环境变量
	term := strings.ToLower(os.Getenv("TERM"))
	sixelTerms := []string{"xterm-sixel", "mlterm", "yaft"}

	for _, sixelTerm := range sixelTerms {
		if strings.Contains(term, sixelTerm) {
			return true
		}
	}

	return false
}

// RenderImage 渲染图片到终端输出
func (r *ImageRenderer) RenderImage(imagePath string, maxWidth, maxHeight int) (string, error) {
	if r.TextMode {
		return r.renderAsANSI(imagePath)
	}
	// 检查文件是否存在
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("image file not found: %s", imagePath),
		}
	}

	// 检查文件格式
	if err := r.validateImageFile(imagePath); err != nil {
		return "", err
	}

	// 使用提供的尺寸或默认尺寸
	width := maxWidth
	height := maxHeight
	if width <= 0 {
		width = r.MaxWidth
	}
	if height <= 0 {
		height = r.MaxHeight
	}

	switch r.Protocol {
	case ProtocolKitty:
		return r.renderWithKitty(imagePath, width, height)
	case ProtocolITerm:
		return r.renderWithITerm(imagePath, width, height)
	case ProtocolSixel:
		return r.renderWithSixel(imagePath, width, height)
	default:
		return r.renderFallback(imagePath)
	}
}

// RenderImageAtCells 使用 Kitty 协议并在指定单元格位置显示（绝对行列）
func (r *ImageRenderer) RenderImageAtCells(imagePath string, cols, rows, startCol, startRow int) (string, error) {
	if r.Protocol != ProtocolKitty {
		// 回退：返回普通渲染，前面自行定位
		return r.RenderImage(imagePath, cols*8, rows*16)
	}

	// 打开并解码图片
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	// 输出到指定绝对位置
	var b strings.Builder
	if startRow > 0 && startCol > 0 {
		fmt.Fprintf(&b, "\x1b[%d;%dH", startRow, startCol)
	}

	// 按目标单元格缩放显示
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	opts := rasterm.KittyImgOpts{
		DstCols: uint32(cols),
		DstRows: uint32(rows),
	}
	if err := rasterm.KittyWriteImage(&b, img, opts); err != nil {
		return "", err
	}
	return b.String(), nil
}

// renderWithKitty 使用 Kitty 协议渲染
func (r *ImageRenderer) renderWithKitty(imagePath string, maxWidth, maxHeight int) (string, error) {
	// 打开图片文件
	file, err := os.Open(imagePath)
	if err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to open image file: %w", err),
		}
	}
	defer file.Close()

	// 解码图片
	img, format, err := image.Decode(file)
	if err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to decode image: %w", err),
		}
	}

	// 预处理图片（调整大小等）
	processedImg, err := r.preprocessImage(img, maxWidth, maxHeight)
	if err != nil {
		return "", err
	}

	// 计算显示尺寸
	bounds := processedImg.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// 计算终端单元格数量
	cols, rows := r.calculateTerminalCells(imgWidth, imgHeight, maxWidth, maxHeight)

	// 创建 Kitty 选项
	opts := rasterm.KittyImgOpts{
		DstCols: cols,
		DstRows: rows,
	}

	// 添加额外的 Kitty 特定选项
	// 注意：rasterm 库的 KittyImgOpts 可能不直接支持动画选项
	_ = format // 暂时忽略格式，避免未使用变量警告

	// 使用 strings.Builder 捕获输出
	var output strings.Builder

	// 使用 rasterm 编码为 Kitty 协议
	if err := rasterm.KittyWriteImage(&output, processedImg, opts); err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to encode image with Kitty protocol: %w", err),
		}
	}

	return output.String(), nil
}

// renderAsANSI 使用 ANSI 24 位颜色半块字符进行纯文本渲染
func (r *ImageRenderer) renderAsANSI(imagePath string) (string, error) {
	// 打开图片
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	// 目标网格
	cols := r.CellCols
	rows := r.CellRows
	if cols <= 0 {
		cols = 40
	}
	if rows <= 0 {
		rows = 12
	}

	// 根据网格计算目标像素尺寸（行*2）
	targetW := cols
	targetH := rows * 2

	// 源尺寸
	sb := img.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	if sw == 0 || sh == 0 {
		return "", fmt.Errorf("invalid image size")
	}

	// 保持比例缩放
	scaleX := float64(targetW) / float64(sw)
	scaleY := float64(targetH) / float64(sh)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}
	outW := int(float64(sw) * scale)
	outH := int(float64(sh) * scale)
	if outW < 1 {
		outW = 1
	}
	if outH < 2 {
		outH = 2
	}

	// 采样函数
	sample := func(x, y int) (uint8, uint8, uint8) {
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		if x >= sw {
			x = sw - 1
		}
		if y >= sh {
			y = sh - 1
		}
		rr, gg, bb, _ := img.At(x, y).RGBA()
		return uint8(rr >> 8), uint8(gg >> 8), uint8(bb >> 8)
	}

	var bldr strings.Builder
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			// 目标像素坐标（缩放后）
			tx := int(float64(col) / float64(cols) * float64(outW))
			tyTop := int(float64(row*2) / float64(rows*2) * float64(outH))
			tyBot := int(float64(row*2+1) / float64(rows*2) * float64(outH))

			// 映射到源图
			sxTop := int(float64(tx) / float64(outW) * float64(sw))
			syTop := int(float64(tyTop) / float64(outH) * float64(sh))
			sxBot := int(float64(tx) / float64(outW) * float64(sw))
			syBot := int(float64(tyBot) / float64(outH) * float64(sh))

			r1, g1, b1 := sample(sxTop, syTop)
			r2, g2, b2 := sample(sxBot, syBot)
			fmt.Fprintf(&bldr, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", r1, g1, b1, r2, g2, b2)
		}
		bldr.WriteString("\x1b[0m\n")
	}
	return bldr.String(), nil
}

// renderWithITerm 使用 iTerm2 协议渲染
func (r *ImageRenderer) renderWithITerm(imagePath string, maxWidth, maxHeight int) (string, error) {
	// 打开图片文件
	file, err := os.Open(imagePath)
	if err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to open image file: %w", err),
		}
	}
	defer file.Close()

	// 解码图片
	img, _, err := image.Decode(file)
	if err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to decode image: %w", err),
		}
	}

	// 预处理图片
	processedImg, err := r.preprocessImage(img, maxWidth, maxHeight)
	if err != nil {
		return "", err
	}

	// 使用 strings.Builder 捕获输出
	var output strings.Builder

	// 使用 rasterm 编码为 iTerm2 协议
	if err := rasterm.ItermWriteImage(&output, processedImg); err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to encode image with iTerm2 protocol: %w", err),
		}
	}

	return output.String(), nil
}

// renderWithSixel 使用 Sixel 协议渲染
func (r *ImageRenderer) renderWithSixel(imagePath string, maxWidth, maxHeight int) (string, error) {
	// 打开图片文件
	file, err := os.Open(imagePath)
	if err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to open image file: %w", err),
		}
	}
	defer file.Close()

	// 解码图片
	img, _, err := image.Decode(file)
	if err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to decode image: %w", err),
		}
	}

	// 预处理图片
	processedImg, err := r.preprocessImage(img, maxWidth, maxHeight)
	if err != nil {
		return "", err
	}

	// 转换为调色板图像（Sixel 需要）
	bounds := processedImg.Bounds()
	palettedImg := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(palettedImg, bounds, processedImg, image.ZP)

	// 使用 strings.Builder 捕获输出
	var output strings.Builder

	// 使用 rasterm 编码为 Sixel 协议
	if err := rasterm.SixelWriteImage(&output, palettedImg); err != nil {
		return "", &RenderError{
			Terminal: string(r.TerminalType),
			Protocol: string(r.Protocol),
			Err:      fmt.Errorf("failed to encode image with Sixel protocol: %w", err),
		}
	}

	return output.String(), nil
}

// renderFallback 降级处理，不支持图形时的文本描述
func (r *ImageRenderer) renderFallback(imagePath string) (string, error) {
	stat, err := os.Stat(imagePath)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("🖼️ Image file: %s (%.1f KB)\n⚠️  Terminal does not support image display",
		filepath.Base(imagePath), float64(stat.Size())/1024.0), nil
}

// IsSupported 检查当前终端是否支持图片渲染
func (r *ImageRenderer) IsSupported() bool {
	return r.Supported && r.Protocol != ProtocolNone
}

// GetCapabilities 获取渲染器能力
func (r *ImageRenderer) GetCapabilities() RendererCapabilities {
	return RendererCapabilities{
		MaxWidth:         r.MaxWidth,
		MaxHeight:        r.MaxHeight,
		SupportedFormats: []ImageFormat{FormatJPEG, FormatPNG, FormatGIF, FormatWebP, FormatBMP},
		HasAlpha:         r.Protocol != ProtocolSixel, // Sixel 对透明度支持有限
		HasAnimation:     r.Protocol == ProtocolKitty, // 只有 Kitty 协议支持动画
	}
}

// SetMaxSize 设置最大渲染尺寸
func (r *ImageRenderer) SetMaxSize(width, height int) {
	if width > 0 {
		if width > MaxImageWidth {
			width = MaxImageWidth
		}
		r.MaxWidth = width
	}
	if height > 0 {
		if height > MaxImageHeight {
			height = MaxImageHeight
		}
		r.MaxHeight = height
	}
}

// SetQuality 设置渲染质量
func (r *ImageRenderer) SetQuality(quality int) {
	if quality >= 1 && quality <= 100 {
		r.Quality = quality
	}
}

// SetTextMode 设置是否使用纯文本 ANSI 渲染
func (r *ImageRenderer) SetTextMode(enabled bool) {
	r.TextMode = enabled
}

// SetCellSize 设置文本模式的单元格尺寸（列、行）
func (r *ImageRenderer) SetCellSize(cols, rows int) {
	if cols > 0 {
		r.CellCols = cols
	}
	if rows > 0 {
		r.CellRows = rows
	}
}

// ClearScreen 清除终端图片显示
func (r *ImageRenderer) ClearScreen() error {
	switch r.Protocol {
	case ProtocolKitty:
		// Kitty 清除命令
		fmt.Print("\033_Ga=d\033\\")
	case ProtocolITerm:
		// iTerm2: 避免全屏清除以免破坏 TUI，这里不执行
		// iTerm2 内联图像不易精确删除，选择不清除，由 Bubble Tea 后续重绘覆盖
	case ProtocolSixel:
		// Sixel: 避免全屏清除
	}
	return nil
}

// validateImageFile 验证图片文件格式
func (r *ImageRenderer) validateImageFile(imagePath string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取文件头
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}

	// 检测内容类型
	contentType := http.DetectContentType(buffer[:n])
	supportedTypes := []string{
		"image/jpeg", "image/png", "image/gif",
		"image/webp", "image/bmp", "image/svg+xml",
	}

	for _, supportedType := range supportedTypes {
		if contentType == supportedType {
			return nil
		}
	}

	return &FormatError{
		Format:   contentType,
		FilePath: imagePath,
		Reason:   "unsupported image format",
	}
}

// preprocessImage 预处理图片（缩放、优化等）
func (r *ImageRenderer) preprocessImage(img image.Image, maxWidth, maxHeight int) (image.Image, error) {
	if !r.AutoResize {
		return img, nil
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 检查是否需要缩放
	if width <= maxWidth && height <= maxHeight {
		return img, nil
	}

	// 计算缩放比例（保持宽高比）
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)

	scale := scaleX
	if r.PreserveAR && scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// 创建新的RGBA图像
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// 简单的最近邻插值缩放
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// 映射到原图坐标
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)

			// 确保坐标在边界内
			if srcX >= width {
				srcX = width - 1
			}
			if srcY >= height {
				srcY = height - 1
			}

			resized.Set(x, y, img.At(srcX, srcY))
		}
	}

	return resized, nil
}

// calculateTerminalCells 计算终端单元格数量
func (r *ImageRenderer) calculateTerminalCells(imgWidth, imgHeight, maxWidth, maxHeight int) (uint32, uint32) {
	// 字符宽高比例 (大约 1:2)
	charRatio := 2.0

	// 计算可用的字符单元
	availableCols := float64(maxWidth) / 8.0   // 假设每个字符8像素宽
	availableRows := float64(maxHeight) / 16.0 // 假设每个字符16像素高

	// 计算图片需要的单元格数
	imgCols := float64(imgWidth) / 8.0
	imgRows := float64(imgHeight) / 16.0 * charRatio

	// 计算缩放比例
	scaleX := availableCols / imgCols
	scaleY := availableRows / imgRows

	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// 限制最小值
	if scale > 1.0 {
		scale = 1.0
	}

	cols := uint32(imgCols * scale)
	rows := uint32(imgRows * scale / charRatio)

	// 确保不为零
	if cols == 0 {
		cols = 1
	}
	if rows == 0 {
		rows = 1
	}

	return cols, rows
}
