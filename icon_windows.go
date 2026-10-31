//go:build windows

package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	_ "image/png"
	"unsafe"

	"gioui.org/app"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/sys/windows"
)

//go:embed assets/icon.png
var iconPNG []byte

var windowIconApplied bool

func applyWindowIcon(e interface{ ImplementsEvent() }) {
	view, ok := e.(app.Win32ViewEvent)
	if !ok || !view.Valid() || windowIconApplied {
		return
	}
	windowIconApplied = true
	hwnd := view.HWND
	go setWindowIcon(hwnd)
}

func setWindowIcon(hwnd uintptr) {
	src, _, err := image.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		return
	}
	big, err := hicon(src, 256)
	if err != nil {
		return
	}
	small, err := hicon(src, 32)
	if err != nil {
		small = big
	}
	postMessage.Call(hwnd, wmSetIcon, iconBig, uintptr(big))
	postMessage.Call(hwnd, wmSetIcon, iconSmall, uintptr(small))
	bigIndex := int32(gclpHIcon)
	smallIndex := int32(gclpHIconSM)
	setClassLong.Call(hwnd, uintptr(bigIndex), uintptr(big))
	setClassLong.Call(hwnd, uintptr(smallIndex), uintptr(small))
}

const (
	wmSetIcon   = 0x0080
	iconSmall   = 0
	iconBig     = 1
	gclpHIcon   = -14
	gclpHIconSM = -34
	biRGB       = 0
	dibRGB      = 0
)

var (
	user32       = windows.NewLazySystemDLL("user32.dll")
	gdi32        = windows.NewLazySystemDLL("gdi32.dll")
	postMessage  = user32.NewProc("PostMessageW")
	setClassLong = user32.NewProc("SetClassLongPtrW")
	getDC        = user32.NewProc("GetDC")
	releaseDC    = user32.NewProc("ReleaseDC")
	createDIB    = gdi32.NewProc("CreateDIBSection")
	createBitmap = gdi32.NewProc("CreateBitmap")
	deleteObject = gdi32.NewProc("DeleteObject")
	createIcon   = user32.NewProc("CreateIconIndirect")
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  windows.Handle
	HbmColor windows.Handle
}

func hicon(src image.Image, size int) (windows.Handle, error) {
	img := scaleNRGBA(src, size)
	colorBmp, err := dibSection(img)
	if err != nil {
		return 0, err
	}
	stride := ((size + 31) / 32) * 4
	maskBits := make([]byte, stride*size)
	mask, _, _ := createBitmap.Call(uintptr(size), uintptr(size), 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))
	info := iconInfo{FIcon: 1, HbmMask: windows.Handle(mask), HbmColor: colorBmp}
	h, _, callErr := createIcon.Call(uintptr(unsafe.Pointer(&info)))
	deleteObject.Call(uintptr(colorBmp))
	if mask != 0 {
		deleteObject.Call(mask)
	}
	if h == 0 {
		return 0, callErr
	}
	return windows.Handle(h), nil
}

func dibSection(img *image.NRGBA) (windows.Handle, error) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	hdr := bitmapInfoHeader{
		Size:        40,
		Width:       int32(w),
		Height:      -int32(h),
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}
	var bits unsafe.Pointer
	hdc, _, _ := getDC.Call(0)
	defer releaseDC.Call(0, hdc)
	hbmp, _, err := createDIB.Call(hdc, uintptr(unsafe.Pointer(&hdr)), dibRGB, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbmp == 0 {
		return 0, err
	}
	pix := unsafe.Slice((*byte)(bits), w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(x, y)
			i := (y*w + x) * 4
			pix[i] = c.B
			pix[i+1] = c.G
			pix[i+2] = c.R
			pix[i+3] = c.A
		}
	}
	return windows.Handle(hbmp), nil
}

func scaleNRGBA(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst
}
