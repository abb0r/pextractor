package main

import (
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/ncruces/zenity"
)

func main() {
	go func() {
		window := new(app.Window)
		window.Option(
			app.Title("pExtractor"),
			app.Size(unit.Dp(980), unit.Dp(680)),
			app.MinSize(unit.Dp(760), unit.Dp(560)),
		)
		if err := run(window); err != nil {
			_ = zenity.Error(err.Error(), zenity.Title("pExtractor"))
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(window *app.Window) error {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette.Fg = colText
	th.Palette.Bg = colBg
	th.Palette.ContrastBg = colAccent
	th.Palette.ContrastFg = colOnAccent

	ui := newUI(window, th)
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			ui.stop()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			ui.layout(gtx)
			e.Frame(gtx.Ops)
		default:
			applyWindowIcon(e)
		}
	}
}
