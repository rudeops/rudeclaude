package kitty

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func Supported(tty *os.File) bool {
	const query = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\\x1b[c"
	if _, err := tty.WriteString(query); err != nil {
		return false
	}
	var got []byte
	buf := make([]byte, 256)
	deadline := time.Now().Add(500 * time.Millisecond)
	for !da1.Match(got) {
		wait := time.Until(deadline)
		if wait <= 0 {
			break
		}
		fds := []unix.PollFd{{Fd: int32(tty.Fd()), Events: unix.POLLIN}}
		if n, err := unix.Poll(fds, int(wait.Milliseconds())+1); err != nil || n == 0 {
			break
		}
		n, err := tty.Read(buf)
		if err != nil {
			break
		}
		got = append(got, buf[:n]...)
	}
	return bytes.Contains(got, []byte("\x1b_Gi=31;OK"))
}

var da1 = regexp.MustCompile(`\x1b\[\?[0-9;]*c`)

type Size struct {
	Cols, Rows   int
	CellW, CellH float64
}

func GetSize(tty *os.File) (Size, error) {
	ws, err := unix.IoctlGetWinsize(int(tty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return Size{}, err
	}
	s := Size{Cols: int(ws.Col), Rows: int(ws.Row)}
	if ws.Xpixel > 0 && ws.Ypixel > 0 && ws.Col > 0 && ws.Row > 0 {
		s.CellW = float64(ws.Xpixel) / float64(ws.Col)
		s.CellH = float64(ws.Ypixel) / float64(ws.Row)
	}
	return s, nil
}

type Placement struct {
	Col, Row int
	OffX     int
	OffY     int
	Z        int
}

func Show(w io.Writer, img *image.RGBA, id int, p Placement) error {
	var z bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&z, zlib.BestSpeed)
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		off := img.PixOffset(b.Min.X, y)
		zw.Write(img.Pix[off : off+b.Dx()*4])
	}
	zw.Close()
	data := base64.StdEncoding.EncodeToString(z.Bytes())

	var out strings.Builder
	fmt.Fprintf(&out, "\x1b[%d;%dH", p.Row, p.Col)
	for i := 0; i < len(data); i += 4096 {
		chunk := data[i:min(i+4096, len(data))]
		more := 0
		if i+4096 < len(data) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(&out, "\x1b_Ga=T,f=32,o=z,s=%d,v=%d,i=%d,p=1,q=2,C=1,X=%d,Y=%d,z=%d,m=%d;%s\x1b\\",
				b.Dx(), b.Dy(), id, p.OffX, p.OffY, p.Z, more, chunk)
		} else {
			fmt.Fprintf(&out, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}
	_, err := io.WriteString(w, out.String())
	return err
}

func Delete(w io.Writer, id int) error {
	_, err := fmt.Fprintf(w, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
	return err
}

func DeleteAll(w io.Writer) error {
	_, err := io.WriteString(w, "\x1b_Ga=d,d=A,q=2\x1b\\")
	return err
}
