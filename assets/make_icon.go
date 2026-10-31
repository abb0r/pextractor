package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

func main() {
	srcFile, err := os.Open("assets/icon.png")
	if err != nil {
		panic(err)
	}
	src, err := png.Decode(srcFile)
	srcFile.Close()
	if err != nil {
		panic(err)
	}
	sizes := []int{16, 32, 48, 256}
	var pngs [][]byte
	for _, size := range sizes {
		dst := image.NewNRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
		var buf bytes.Buffer
		if err := png.Encode(&buf, dst); err != nil {
			panic(err)
		}
		pngs = append(pngs, buf.Bytes())
		if size == 128 || size == 256 || size == 512 {
			continue
		}
	}
	for _, size := range []int{128, 256, 512} {
		dst := image.NewNRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
		f, err := os.Create("assets/icon-" + itoa(size) + ".png")
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, dst); err != nil {
			panic(err)
		}
		f.Close()
	}
	if err := os.WriteFile("assets/icon.ico", icoPNG(sizes, pngs), 0o644); err != nil {
		panic(err)
	}
}

func icoPNG(sizes []int, pngs [][]byte) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(pngs)))
	offset := 6 + 16*len(pngs)
	for i, size := range sizes {
		w := byte(size)
		if size >= 256 {
			w = 0
		}
		buf.WriteByte(w)
		buf.WriteByte(w)
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pngs[i])))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		buf.Write(p)
	}
	return buf.Bytes()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
