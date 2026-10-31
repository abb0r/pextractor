package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"pextractor/plex"
)

var (
	colBg       = color.NRGBA{R: 0x0D, G: 0x0D, B: 0x12, A: 0xFF}
	colSurface  = color.NRGBA{R: 0x17, G: 0x17, B: 0x20, A: 0xFF}
	colBorder   = color.NRGBA{R: 0x2E, G: 0x2E, B: 0x3C, A: 0xFF}
	colText     = color.NRGBA{R: 0xF2, G: 0xF2, B: 0xF7, A: 0xFF}
	colMuted    = color.NRGBA{R: 0xA4, G: 0xA4, B: 0xB8, A: 0xFF}
	colHint     = color.NRGBA{R: 0x7E, G: 0x7E, B: 0x96, A: 0xFF}
	colAccent   = color.NRGBA{R: 0x6B, G: 0x63, B: 0xF6, A: 0xFF}
	colOnAccent = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	colSelect   = color.NRGBA{R: 0x2A, G: 0x27, B: 0x4E, A: 0xFF}
	colHover    = color.NRGBA{R: 0x22, G: 0x22, B: 0x30, A: 0xFF}
	colError    = color.NRGBA{R: 0xFF, G: 0x8A, B: 0x80, A: 0xFF}
	colDisabled = color.NRGBA{R: 0x3A, G: 0x3A, B: 0x4A, A: 0xFF}
)

type ui struct {
	win *app.Window
	th  *material.Theme

	server    widget.Editor
	token     widget.Editor
	showToken widget.Bool
	insecure  widget.Bool
	connect   widget.Clickable
	export    widget.Clickable
	libList   widget.List
	lineList  widget.List
	libClicks []widget.Clickable

	mu           sync.Mutex
	status       string
	statusErr    bool
	busy         bool
	summary      string
	showProgress bool
	progress     float32
	libraries    []plex.Library
	selected     int
	lines        []string
	serverURL    string
	client       *plex.Client
	cancel       context.CancelFunc
	req          int64
	saved        connectionConfig
}

type snapshot struct {
	status       string
	statusErr    bool
	busy         bool
	summary      string
	showProgress bool
	progress     float32
	libraries    []plex.Library
	selected     int
	lines        []string
}

func newUI(win *app.Window, th *material.Theme) *ui {
	u := &ui{win: win, th: th, selected: -1}
	u.server.SingleLine = true
	u.server.Submit = true
	u.token.SingleLine = true
	u.token.Submit = true
	u.token.Mask = '•'
	u.libList.Axis = layout.Vertical
	u.lineList.Axis = layout.Vertical
	u.loadConfig()
	return u
}

func (u *ui) stop() {
	u.persistConfig()
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.cancel != nil {
		u.cancel()
	}
}

func (u *ui) snapshot() snapshot {
	u.mu.Lock()
	defer u.mu.Unlock()
	return snapshot{
		status:       u.status,
		statusErr:    u.statusErr,
		busy:         u.busy,
		summary:      u.summary,
		showProgress: u.showProgress,
		progress:     u.progress,
		libraries:    u.libraries,
		selected:     u.selected,
		lines:        u.lines,
	}
}

func (u *ui) layout(gtx layout.Context) layout.Dimensions {
	if u.showToken.Value {
		u.token.Mask = 0
	} else {
		u.token.Mask = '•'
	}
	submit := editorSubmitted(gtx, &u.server) || editorSubmitted(gtx, &u.token)
	connectClick := u.connect.Clicked(gtx)
	exportClick := u.export.Clicked(gtx)
	view := u.snapshot()
	if len(u.libClicks) != len(view.libraries) {
		u.libClicks = make([]widget.Clickable, len(view.libraries))
	}
	picked := -1
	for i := range u.libClicks {
		if u.libClicks[i].Clicked(gtx) {
			picked = i
		}
	}

	paint.FillShape(gtx.Ops, colBg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	dims := layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return u.column(gtx, view)
	})

	if submit || connectClick {
		u.connectServer()
	} else if picked >= 0 {
		u.openLibrary(picked)
	}
	if exportClick {
		u.exportLines()
	}
	u.persistConfig()
	return dims
}

func (u *ui) column(gtx layout.Context, view snapshot) layout.Dimensions {
	side := u.sideWidth(gtx)
	gap := gtx.Dp(unit.Dp(20))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.H5(u.th, "pExtractor")
			l.Color = colText
			return l.Layout(gtx)
		}),
		layout.Rigid(u.gap(2)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.muted(gtx, unit.Sp(13), "Movie and series titles, and artists from music libraries.")
		}),
		layout.Rigid(u.gap(16)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = side
					gtx.Constraints.Max.X = side
					gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
					return u.connection(gtx, view)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(20)}.Layout(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
					return u.results(gtx, view)
				}),
			)
		}),
		layout.Rigid(u.gap(12)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(side+gap, 1)}
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return u.footer(gtx, view)
				}),
			)
		}),
	)
}

func (u *ui) sideWidth(gtx layout.Context) int {
	w := gtx.Dp(unit.Dp(340))
	if max := gtx.Constraints.Max.X; w > max/2 {
		w = max / 2
	}
	return w
}

func (u *ui) connection(gtx layout.Context, view snapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.field(gtx, "Server", "http://192.168.1.10:32400", &u.server)
		}),
		layout.Rigid(u.gap(12)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.field(gtx, "Token", "Plex token", &u.token)
		}),
		layout.Rigid(u.gap(6)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.muted(gtx, unit.Sp(12), "In Plex Web, open a title, then Get Info and View XML. The token is the X-Plex-Token value in the address.")
		}),
		layout.Rigid(u.gap(2)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(u.th, &u.showToken, "Show token").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(u.th, &u.insecure, "Allow self-signed certificates").Layout(gtx)
		}),
		layout.Rigid(u.gap(12)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := "Connect"
			if view.busy && len(view.libraries) == 0 {
				label = "Connecting…"
			}
			return u.button(gtx, &u.connect, label, !view.busy)
		}),
		layout.Rigid(u.gap(16)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.caption(gtx, "Library")
		}),
		layout.Rigid(u.gap(6)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return u.panel(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
				if len(view.libraries) == 0 {
					return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return u.muted(gtx, unit.Sp(14), "No libraries loaded yet.")
					})
				}
				return material.List(u.th, &u.libList).Layout(gtx, len(view.libraries), func(gtx layout.Context, i int) layout.Dimensions {
					return u.libraryRow(gtx, view, i)
				})
			})
		}),
	)
}

func (u *ui) results(gtx layout.Context, view snapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			text := view.summary
			if text == "" {
				text = "Entries"
			}
			return u.caption(gtx, text)
		}),
		layout.Rigid(u.gap(6)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !view.showProgress {
				return layout.Dimensions{}
			}
			bar := material.ProgressBar(u.th, view.progress)
			bar.Color = colAccent
			bar.TrackColor = colBorder
			bar.Height = unit.Dp(4)
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, bar.Layout)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return u.panel(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
				if len(view.lines) == 0 {
					msg := "Choose a library."
					if len(view.libraries) == 0 {
						msg = "Connect to the Plex server."
					}
					return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return u.muted(gtx, unit.Sp(14), msg)
					})
				}
				return material.List(u.th, &u.lineList).Layout(gtx, len(view.lines), func(gtx layout.Context, i int) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, unit.Sp(14), view.lines[i])
						l.Color = colText
						l.MaxLines = 1
						return l.Layout(gtx)
					})
				})
			})
		}),
	)
}

func (u *ui) footer(gtx layout.Context, view snapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.button(gtx, &u.export, "Export as TXT", len(view.lines) > 0 && !view.busy)
		}),
		layout.Rigid(u.gap(8)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if view.status == "" {
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(18)))}
			}
			l := material.Label(u.th, unit.Sp(13), view.status)
			l.Color = colMuted
			if view.statusErr {
				l.Color = colError
			}
			return l.Layout(gtx)
		}),
	)
}

func (u *ui) libraryRow(gtx layout.Context, view snapshot, i int) layout.Dimensions {
	lib := view.libraries[i]
	return u.libClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		bg := colSurface
		switch {
		case i == view.selected:
			bg = colSelect
		case u.libClicks[i].Hovered():
			bg = colHover
		}
		return fill(gtx, bg, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, unit.Sp(14), lib.Title)
						l.Color = colText
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.muted(gtx, unit.Sp(12), typeLabel(lib.Type))
					}),
				)
			})
		})
	})
}

func (u *ui) field(gtx layout.Context, caption, hint string, ed *widget.Editor) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.caption(gtx, caption)
		}),
		layout.Rigid(u.gap(6)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := gtx.Dp(unit.Dp(42))
			gtx.Constraints.Min.Y = h
			gtx.Constraints.Max.Y = h
			return u.panel(gtx, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(9), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					e := material.Editor(u.th, ed, hint)
					e.TextSize = unit.Sp(15)
					e.Color = colText
					e.HintColor = colHint
					e.SelectionColor = color.NRGBA{R: 0x6B, G: 0x63, B: 0xF6, A: 0x66}
					return e.Layout(gtx)
				})
			})
		}),
	)
}

func (u *ui) button(gtx layout.Context, click *widget.Clickable, text string, enabled bool) layout.Dimensions {
	b := material.Button(u.th, click, text)
	b.CornerRadius = 8
	b.TextSize = unit.Sp(14)
	b.Inset = layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(16), Right: unit.Dp(16)}
	b.Background = colAccent
	b.Color = colOnAccent
	if !enabled {
		b.Background = colDisabled
	}
	return b.Layout(gtx)
}

func (u *ui) panel(gtx layout.Context, radius unit.Dp, body layout.Widget) layout.Dimensions {
	size := gtx.Constraints.Max
	if size.X < gtx.Constraints.Min.X {
		size.X = gtx.Constraints.Min.X
	}
	if size.Y < gtx.Constraints.Min.Y {
		size.Y = gtx.Constraints.Min.Y
	}
	rr := func(rect image.Rectangle, rad int) clip.Op {
		if rad < 0 {
			rad = 0
		}
		return clip.RRect{Rect: rect, NE: rad, NW: rad, SE: rad, SW: rad}.Op(gtx.Ops)
	}
	rad := gtx.Dp(radius)
	paint.FillShape(gtx.Ops, colBorder, rr(image.Rectangle{Max: size}, rad))
	inset := gtx.Dp(unit.Dp(1))
	inner := image.Rectangle{Min: image.Pt(inset, inset), Max: size.Sub(image.Pt(inset, inset))}
	if inner.Dx() > 0 && inner.Dy() > 0 {
		paint.FillShape(gtx.Ops, colSurface, rr(inner, rad-inset))
	}
	defer clip.RRect{Rect: image.Rectangle{Max: size}, NE: rad, NW: rad, SE: rad, SW: rad}.Push(gtx.Ops).Pop()
	innerGtx := gtx
	innerGtx.Constraints = layout.Exact(size)
	_ = layout.UniformInset(unit.Dp(1)).Layout(innerGtx, body)
	return layout.Dimensions{Size: size}
}

func (u *ui) caption(gtx layout.Context, text string) layout.Dimensions {
	l := material.Label(u.th, unit.Sp(13), text)
	l.Color = colMuted
	l.MaxLines = 1
	return l.Layout(gtx)
}

func (u *ui) muted(gtx layout.Context, size unit.Sp, text string) layout.Dimensions {
	l := material.Label(u.th, size, text)
	l.Color = colMuted
	return l.Layout(gtx)
}

func (u *ui) gap(h unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Height: h}.Layout(gtx)
	}
}

func fill(gtx layout.Context, c color.NRGBA, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	paint.FillShape(gtx.Ops, c, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

func editorSubmitted(gtx layout.Context, ed *widget.Editor) bool {
	submitted := false
	for {
		ev, ok := ed.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			submitted = true
		}
	}
	return submitted
}

func (u *ui) connectServer() {
	base, token, err := normalize(u.server.Text(), u.token.Text())
	if err != nil {
		u.setStatus(err.Error(), true)
		return
	}
	insecure := u.insecure.Value
	u.server.SetText(base)
	if strings.TrimSpace(u.token.Text()) == "" {
		u.token.SetText(token)
	}
	ctx, id := u.begin(func() {
		u.libraries = nil
		u.lines = nil
		u.selected = -1
		u.summary = ""
		u.client = nil
		u.serverURL = ""
		u.status = "Connecting…"
		u.statusErr = false
		u.showProgress = false
	})
	go func() {
		client, err := plex.NewClient(base, token, insecure)
		if err != nil {
			u.complete(id, err, nil)
			return
		}
		libs, err := client.Libraries(ctx)
		u.complete(id, err, func() {
			u.client = client
			u.serverURL = base
			u.libraries = libs
			u.summary = libraryCount(len(libs))
			u.status = "Select a library."
			u.statusErr = false
		})
	}()
}

func (u *ui) openLibrary(index int) {
	u.mu.Lock()
	if index < 0 || index >= len(u.libraries) || u.client == nil {
		u.mu.Unlock()
		return
	}
	lib := u.libraries[index]
	client := u.client
	u.mu.Unlock()

	if !lib.Supported() {
		u.mu.Lock()
		if u.cancel != nil {
			u.cancel()
			u.cancel = nil
		}
		u.req++
		u.busy = false
		u.selected = index
		u.lines = nil
		u.summary = lib.Title
		u.showProgress = false
		u.status = unsupportedMessage(lib.Type)
		u.statusErr = false
		u.mu.Unlock()
		u.win.Invalidate()
		return
	}

	ctx, id := u.begin(func() {
		u.selected = index
		u.lines = nil
		u.summary = lib.Title
		u.status = "Loading entries…"
		u.statusErr = false
		u.showProgress = false
		u.progress = 0
	})
	go func() {
		lines, err := client.Entries(ctx, lib, func(loaded, total int) {
			u.mu.Lock()
			defer u.mu.Unlock()
			if u.req != id {
				return
			}
			if total > 0 {
				u.showProgress = true
				u.progress = float32(loaded) / float32(total)
				u.status = fmt.Sprintf("Loading entries… %d/%d", loaded, total)
			} else {
				u.status = fmt.Sprintf("Loading entries… %d", loaded)
			}
			u.win.Invalidate()
		})
		u.complete(id, err, func() {
			u.lines = lines
			u.summary = lib.Title + " · " + countLabel(len(lines), lib.Type)
			u.status = ""
			if len(lines) == 0 {
				u.status = "No entries in this library."
			}
			u.statusErr = false
		})
	}()
}

func (u *ui) exportLines() {
	u.mu.Lock()
	if u.busy || len(u.lines) == 0 {
		busy := u.busy
		u.mu.Unlock()
		if !busy {
			u.setStatus("Nothing to export.", true)
		}
		return
	}
	doc := exportDoc{
		Server: u.serverURL,
		Lines:  append([]string(nil), u.lines...),
	}
	if u.selected >= 0 && u.selected < len(u.libraries) {
		lib := u.libraries[u.selected]
		doc.Library = lib.Title
		doc.Kind = typeLabel(lib.Type)
	}
	u.mu.Unlock()

	path, err := saveExport(doc)
	if err != nil {
		u.setStatus(err.Error(), true)
		return
	}
	if path == "" {
		return
	}
	u.setStatus("Saved: "+path, false)
}

func (u *ui) begin(setup func()) (context.Context, int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.cancel != nil {
		u.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = cancel
	u.req++
	u.busy = true
	if setup != nil {
		setup()
	}
	u.win.Invalidate()
	return ctx, u.req
}

func (u *ui) complete(id int64, err error, apply func()) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if id != u.req {
		return
	}
	u.busy = false
	u.showProgress = false
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		u.status = err.Error()
		u.statusErr = true
		u.win.Invalidate()
		return
	}
	if apply != nil {
		apply()
	}
	u.win.Invalidate()
}

func (u *ui) setStatus(msg string, isErr bool) {
	u.mu.Lock()
	u.status = msg
	u.statusErr = isErr
	u.mu.Unlock()
	u.win.Invalidate()
}

func libraryCount(n int) string {
	if n == 1 {
		return "1 library"
	}
	return itoa(n) + " libraries"
}

func unsupportedMessage(kind string) string {
	switch kind {
	case "photo":
		return "Photo libraries are not read."
	default:
		return "This library type is not supported. Movies, TV shows, and music are supported."
	}
}
