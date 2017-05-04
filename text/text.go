package text

import (
	"image/color"
	"math"
	"unicode"
	"unicode/utf8"

	"github.com/faiface/pixel"
	"golang.org/x/image/font"
)

var ASCII []rune

func init() {
	ASCII = make([]rune, unicode.MaxASCII-32)
	for i := range ASCII {
		ASCII[i] = rune(32 + i)
	}
}

func RangeTable(table *unicode.RangeTable) []rune {
	var runes []rune
	for _, rng := range table.R16 {
		for r := rng.Lo; r <= rng.Hi; r += rng.Stride {
			runes = append(runes, rune(r))
		}
	}
	for _, rng := range table.R32 {
		for r := rng.Lo; r <= rng.Hi; r += rng.Stride {
			runes = append(runes, rune(r))
		}
	}
	return runes
}

type Text struct {
	Orig pixel.Vec
	Dot  pixel.Vec

	atlas *Atlas

	lineHeight float64
	tabWidth   float64

	prevR rune
	glyph pixel.TrianglesData
	tris  pixel.TrianglesData
	d     pixel.Drawer
	trans *pixel.Batch
}

func New(face font.Face, runeSets ...[]rune) *Text {
	runes := []rune{unicode.ReplacementChar}
	for _, set := range runeSets {
		runes = append(runes, set...)
	}

	atlas := NewAtlas(face, runes)

	txt := &Text{
		atlas:      atlas,
		lineHeight: atlas.LineHeight(),
		tabWidth:   atlas.Glyph(' ').Advance * 4,
	}

	txt.glyph.SetLen(6)
	for i := range txt.glyph {
		txt.glyph[i].Color = pixel.Alpha(1)
		txt.glyph[i].Intensity = 1
	}

	txt.d.Picture = txt.atlas.pic
	txt.d.Triangles = &txt.tris
	txt.trans = pixel.NewBatch(&pixel.TrianglesData{}, atlas.pic)

	txt.Clear()

	return txt
}

func (txt *Text) Atlas() *Atlas {
	return txt.atlas
}

func (txt *Text) SetMatrix(m pixel.Matrix) {
	txt.trans.SetMatrix(m)
}

func (txt *Text) SetColorMask(c color.Color) {
	txt.trans.SetColorMask(c)
}

func (txt *Text) Color(c color.Color) {
	rgba := pixel.ToRGBA(c)
	for i := range txt.glyph {
		txt.glyph[i].Color = rgba
	}
}

func (txt *Text) LineHeight(scale float64) {
	txt.lineHeight = scale
}

func (txt *Text) TabWidth(width float64) {
	txt.tabWidth = width
}

func (txt *Text) Clear() {
	txt.prevR = -1
	txt.tris.SetLen(0)
	txt.d.Dirty()
}

func (txt *Text) Write(p []byte) (n int, err error) {
	n, err = len(p), nil // always returns this

	if len(p) == 0 {
		return
	}

	for len(p) > 0 {
		r, size := utf8.DecodeRune(p)
		p = p[size:]
		txt.WriteRune(r)
	}

	return
}

func (txt *Text) WriteString(s string) (n int, err error) {
	if len(s) == 0 {
		return
	}

	for _, r := range s {
		txt.WriteRune(r)
	}

	return len(s), nil
}

func (txt *Text) WriteByte(c byte) error {
	_, err := txt.WriteRune(rune(c))
	return err
}

func (txt *Text) WriteRune(r rune) (n int, err error) {
	n, err = utf8.RuneLen(r), nil // always returns this

	switch r {
	case '\n':
		txt.Dot -= pixel.Y(txt.lineHeight)
		txt.Dot = txt.Dot.WithX(txt.Orig.X())
		return
	case '\r':
		txt.Dot = txt.Dot.WithX(txt.Orig.X())
		return
	case '\t':
		rem := math.Mod(txt.Dot.X()-txt.Orig.X(), txt.tabWidth)
		rem = math.Mod(rem, rem+txt.tabWidth)
		if rem == 0 {
			rem = txt.tabWidth
		}
		txt.Dot += pixel.X(rem)
		return
	}

	if !txt.atlas.Contains(r) {
		r = unicode.ReplacementChar
	}
	if !txt.atlas.Contains(unicode.ReplacementChar) {
		return
	}

	glyph := txt.atlas.Glyph(r)

	if txt.prevR >= 0 {
		txt.Dot += pixel.X(txt.atlas.Kern(txt.prevR, r))
	}

	a := pixel.V(glyph.Frame.Min.X(), glyph.Frame.Min.Y())
	b := pixel.V(glyph.Frame.Max.X(), glyph.Frame.Min.Y())
	c := pixel.V(glyph.Frame.Max.X(), glyph.Frame.Max.Y())
	d := pixel.V(glyph.Frame.Min.X(), glyph.Frame.Max.Y())

	for i, v := range []pixel.Vec{a, b, c, a, c, d} {
		txt.glyph[i].Position = v - glyph.Orig + txt.Dot
		txt.glyph[i].Picture = v
	}

	txt.tris = append(txt.tris, txt.glyph...)

	txt.Dot += pixel.X(glyph.Advance)
	txt.prevR = r

	txt.d.Dirty()

	return
}

func (txt *Text) Draw(t pixel.Target) {
	txt.trans.Clear()
	txt.d.Draw(txt.trans)
	txt.trans.Draw(t)
}