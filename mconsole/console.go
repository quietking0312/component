// Package mconsole 提供跨平台终端颜色、样式输出支持（Windows/Linux）。
// Windows 下通过 Virtual Terminal Processing 启用 ANSI 支持，无需第三方依赖。
package mconsole

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ========== ANSI 属性码 ==========

// Style 文字样式
type Style int

const (
	StyleReset     Style = 0 // 重置所有样式
	StyleBold      Style = 1 // 粗体
	StyleDim       Style = 2 // 暗淡（降低亮度）
	StyleItalic    Style = 3 // 斜体
	StyleUnderline Style = 4 // 下划线
	StyleBlink     Style = 5 // 闪烁
	StyleReverse   Style = 7 // 反色（前景色与背景色互换）
	StyleStrike    Style = 9 // 删除线
)

// Color 前景色
type Color int

const (
	ColorDefault Color = -1 // 终端默认前景色
	ColorBlack   Color = 30 // 黑色
	ColorRed     Color = 31 // 红色
	ColorGreen   Color = 32 // 绿色
	ColorYellow  Color = 33 // 黄色
	ColorBlue    Color = 34 // 蓝色
	ColorMagenta Color = 35 // 品红（洋红）
	ColorCyan    Color = 36 // 青色
	ColorWhite   Color = 37 // 白色

	// 高亮（亮色）版本
	ColorBrightBlack   Color = 90 // 亮黑（深灰）
	ColorBrightRed     Color = 91 // 亮红
	ColorBrightGreen   Color = 92 // 亮绿
	ColorBrightYellow  Color = 93 // 亮黄
	ColorBrightBlue    Color = 94 // 亮蓝
	ColorBrightMagenta Color = 95 // 亮品红
	ColorBrightCyan    Color = 96 // 亮青
	ColorBrightWhite   Color = 97 // 亮白
)

// BgColor 背景色（前景色值 +10）
type BgColor int

const (
	BgDefault Color = -1 // 终端默认背景色
	BgBlack   Color = 40 // 黑色背景
	BgRed     Color = 41 // 红色背景
	BgGreen   Color = 42 // 绿色背景
	BgYellow  Color = 43 // 黄色背景
	BgBlue    Color = 44 // 蓝色背景
	BgMagenta Color = 45 // 品红背景
	BgCyan    Color = 46 // 青色背景
	BgWhite   Color = 47 // 白色背景

	BgBrightBlack   Color = 100 // 亮黑背景（深灰）
	BgBrightRed     Color = 101 // 亮红背景
	BgBrightGreen   Color = 102 // 亮绿背景
	BgBrightYellow  Color = 103 // 亮黄背景
	BgBrightBlue    Color = 104 // 亮蓝背景
	BgBrightMagenta Color = 105 // 亮品红背景
	BgBrightCyan    Color = 106 // 亮青背景
	BgBrightWhite   Color = 107 // 亮白背景
)

// ========== 构建 ANSI 序列 ==========

// Render 将文字包裹在 ANSI 转义序列中。
// fg/bg 传 -1 表示不设置颜色；styles 可叠加多个。
func Render(text string, fg Color, bg Color, styles ...Style) string {
	codes := make([]string, 0, 4)
	for _, s := range styles {
		codes = append(codes, fmt.Sprintf("%d", s))
	}
	if fg >= 0 {
		codes = append(codes, fmt.Sprintf("%d", fg))
	}
	if bg >= 0 {
		codes = append(codes, fmt.Sprintf("%d", bg))
	}
	if len(codes) == 0 {
		return text
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", strings.Join(codes, ";"), text)
}

// RGB 使用24位真彩色渲染前景色（终端需支持）。
func RGB(text string, r, g, b uint8) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

// RGBBg 使用24位真彩色渲染背景色。
func RGBBg(text string, r, g, b uint8) string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

// ========== 快捷函数 ==========

func Bold(text string) string      { return Render(text, -1, -1, StyleBold) }
func Dim(text string) string       { return Render(text, -1, -1, StyleDim) }
func Italic(text string) string    { return Render(text, -1, -1, StyleItalic) }
func Underline(text string) string { return Render(text, -1, -1, StyleUnderline) }
func Blink(text string) string     { return Render(text, -1, -1, StyleBlink) }
func Strike(text string) string    { return Render(text, -1, -1, StyleStrike) }
func Reverse(text string) string   { return Render(text, -1, -1, StyleReverse) }

func Red(text string) string     { return Render(text, ColorRed, -1) }
func Green(text string) string   { return Render(text, ColorGreen, -1) }
func Yellow(text string) string  { return Render(text, ColorYellow, -1) }
func Blue(text string) string    { return Render(text, ColorBlue, -1) }
func Magenta(text string) string { return Render(text, ColorMagenta, -1) }
func Cyan(text string) string    { return Render(text, ColorCyan, -1) }
func White(text string) string   { return Render(text, ColorWhite, -1) }

func BoldRed(text string) string    { return Render(text, ColorRed, -1, StyleBold) }
func BoldGreen(text string) string  { return Render(text, ColorGreen, -1, StyleBold) }
func BoldYellow(text string) string { return Render(text, ColorYellow, -1, StyleBold) }
func BoldBlue(text string) string   { return Render(text, ColorBlue, -1, StyleBold) }
func BoldCyan(text string) string   { return Render(text, ColorCyan, -1, StyleBold) }

// ========== Console 写入器 ==========

// Console 封装输出目标，支持自动检测是否为终端（非终端时剥离 ANSI）。
type Console struct {
	w       io.Writer
	noColor bool // 强制禁用颜色
}

// New 创建 Console，输出到 os.Stdout，自动检测终端。
func New() *Console {
	return &Console{w: os.Stdout, noColor: !isTerminal(os.Stdout)}
}

// NewWriter 创建 Console，输出到指定 writer。
func NewWriter(w io.Writer) *Console {
	f, ok := w.(*os.File)
	noColor := ok && !isTerminal(f)
	return &Console{w: w, noColor: noColor}
}

func (c *Console) Color() *Console {
	c.noColor = false
	return c
}

// NoColor 强制禁用颜色输出（适用于日志文件等场景）。
func (c *Console) NoColor() *Console {
	c.noColor = true
	return c
}

// strip 剥离 ANSI 序列（非终端输出时使用）。
func (c *Console) strip(s string) string {
	if !c.noColor {
		return s
	}
	// 简单状态机剥离 \x1b[...m
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip 'm'
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func (c *Console) write(s string) {
	fmt.Fprint(c.w, c.strip(s))
}

// Print 输出已渲染的字符串。
func (c *Console) Print(s string) *Console {
	c.write(s)
	return c
}

// Println 输出并换行。
func (c *Console) Println(s string) *Console {
	c.write(s + "\n")
	return c
}

// Printf 格式化输出。
func (c *Console) Printf(format string, a ...any) *Console {
	c.write(fmt.Sprintf(format, a...))
	return c
}
