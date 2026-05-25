package mconsole_test

import (
	"testing"

	"github.com/quietking0312/component/mconsole"
)

func TestRender(t *testing.T) {
	c := mconsole.New().Color()

	c.Println(mconsole.Bold("粗体"))
	c.Println(mconsole.Underline("下划线"))
	c.Println(mconsole.Strike("删除线"))
	c.Println(mconsole.Italic("斜体"))
	c.Println(mconsole.Blink("闪烁"))
	c.Println(mconsole.Reverse("反色"))
	c.Println("")

	c.Println(mconsole.Red("红色"))
	c.Println(mconsole.Green("绿色"))
	c.Println(mconsole.Yellow("黄色"))
	c.Println(mconsole.Blue("蓝色"))
	c.Println(mconsole.Magenta("紫色"))
	c.Println(mconsole.Cyan("青色"))
	c.Println(mconsole.White("白色"))
	c.Println("")

	c.Println(mconsole.BoldRed("粗体红"))
	c.Println(mconsole.BoldGreen("粗体绿"))
	c.Println(mconsole.BoldYellow("粗体黄"))
	c.Println(mconsole.BoldBlue("粗体蓝"))
	c.Println(mconsole.BoldCyan("粗体青"))
	c.Println("")

	// 自定义组合：绿色背景 + 黑色前景 + 下划线
	c.Println(mconsole.Render("自定义组合", mconsole.ColorBlack, mconsole.BgGreen, mconsole.StyleUnderline))

	// 高亮色
	c.Println(mconsole.Render("亮红高亮", mconsole.ColorBrightRed, -1, mconsole.StyleBold))

	// 24位真彩色
	c.Println(mconsole.RGB("橙色 RGB(255,165,0)", 255, 165, 0))
	c.Println(mconsole.RGBBg("蓝色背景", 30, 144, 255))
}
