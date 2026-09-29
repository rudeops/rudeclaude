package ui

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/signal"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/rudeops/rudeclaude/internal/activity"
	"github.com/rudeops/rudeclaude/internal/gfx"
	"github.com/rudeops/rudeclaude/internal/kitty"
	"github.com/rudeops/rudeclaude/internal/rtk"
)

const imageFPS = 6

func GraphicsSupported() bool {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return false
	}
	defer term.Restore(fd, state)
	return kitty.Supported(os.Stdin)
}

type layout struct {
	scale    float64
	at       kitty.Placement
	tooSmall bool
	link     string
}

func computeLayout(sz kitty.Size) layout {
	cw, ch := sz.CellW, sz.CellH
	if cw == 0 || ch == 0 {
		cw, ch = 9, 18
	}
	pxW, pxH := float64(sz.Cols)*cw, float64(sz.Rows)*ch
	scale := math.Min(2.5, math.Min(pxW*0.96/gfx.OverviewW, pxH*0.96/gfx.OverviewH))
	x := math.Floor((pxW - math.Round(gfx.OverviewW*scale)) / 2)
	y := math.Floor((pxH - math.Round(gfx.OverviewH*scale)) / 2)
	col, row := math.Floor(x/cw), math.Floor(y/ch)
	return layout{
		scale: scale,
		at: kitty.Placement{
			Col: int(col) + 1, Row: int(row) + 1,
			OffX: int(x - col*cw), OffY: int(y - row*ch),
		},
		tooSmall: scale < 0.55,
		link:     signatureLink(x, y, scale, cw, ch),
	}
}

func signatureLink(x, y, scale, cw, ch float64) string {
	sx, sy, sw, sh := gfx.SignatureRect()
	c0 := int(math.Floor((x+sx*scale)/cw)) + 1
	c1 := int(math.Ceil((x + (sx+sw)*scale) / cw))
	row := int(math.Floor((y+(sy+sh/2)*scale)/ch)) + 1
	return fmt.Sprintf("\x1b[%d;%dH\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
		row, c0, gfx.SignatureURL, strings.Repeat(" ", max(1, c1-c0+1)))
}

type buffered struct {
	cur  int
	live bool
}

func (b *buffered) show(w *bufio.Writer, img *image.RGBA, p kitty.Placement) {
	next := 1 - b.cur
	kitty.Show(w, img, next+1, p)
	if b.live {
		kitty.Delete(w, b.cur+1)
	}
	b.cur, b.live = next, true
}

func RunImage(interval time.Duration, demo bool) error {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, state)

	w := bufio.NewWriterSize(os.Stdout, 1<<20)
	fmt.Fprint(w, "\x1b[?1049h\x1b[?25l\x1b[2J")
	w.Flush()
	defer func() {
		kitty.DeleteAll(w)
		fmt.Fprint(w, "\x1b[?25h\x1b[?1049l")
		w.Flush()
	}()

	keys := make(chan byte, 16)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				close(keys)
				return
			}
			for _, k := range buf[:n] {
				keys <- k
			}
		}
	}()
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(quit)

	c := newCore(interval, demo)
	reports := make(chan reportMsg, 1)
	fetch := func(maxAge time.Duration) {
		c.loading = true
		go func() { reports <- fetchReport(maxAge) }()
	}
	if !demo {
		fetch(interval)
	}
	c.refresh()

	var (
		lay    layout
		screen buffered
		shown  gfx.Overview
		frames = time.NewTicker(time.Second / imageFPS)
		polls  = time.NewTicker(pollInterval)
		resize = true
	)
	defer frames.Stop()
	defer polls.Stop()

	for {
		select {
		case k, ok := <-keys:
			if !ok || k == 'q' || k == 3 {
				return nil
			}
			if k == 'r' && !c.loading && !demo {
				fetch(0)
			}
		case <-winch:
			resize = true
		case <-quit:
			return nil
		case msg := <-reports:
			c.setReport(msg)
		case <-polls.C:
			c.refresh()
		case t := <-frames.C:
			c.now = t
			if c.fetchDue() {
				fetch(interval)
			}
			if resize {
				sz, err := kitty.GetSize(os.Stdin)
				if err != nil {
					return err
				}
				lay = computeLayout(sz)
				kitty.DeleteAll(w)
				fmt.Fprint(w, "\x1b[2J")
				screen.live, resize = false, false
				if lay.tooSmall {
					fmt.Fprint(w, "\x1b[1;1Hagrandis la fenêtre pour afficher rudeclaude · q pour quitter")
				} else {
					fmt.Fprint(w, lay.link)
				}
			}
			if o := c.overview(); !lay.tooSmall && (!screen.live || !reflect.DeepEqual(o, shown)) {
				img := gfx.RenderOverview(o, gfx.Options{Scale: lay.scale, Transparent: true})
				screen.show(w, img, lay.at)
				shown = o
			}
			w.Flush()
		}
	}
}

func (c *core) overview() gfx.Overview {
	o := gfx.Overview{
		Updated:     c.status(),
		ActivityNow: c.activityText(),
		Activity:    slices.Clone(c.snap.Minutes[:]),
		Credits:     c.credits(),
	}
	o.Alert = c.alert()
	for i, l := range c.limits() {
		elapsed := math.Round(l.elapsed*1000) / 1000
		o.Limits[i] = gfx.Limit{Label: l.label, Used: l.used, Elapsed: elapsed, Reset: l.reset}
	}
	replies, output, cache := c.today()
	o.Today = []gfx.Stat{
		{Value: replies, Label: "réponses"},
		{Value: output, Label: "tokens générés"},
		{Value: cache, Unit: "%", Label: "servis par le cache"},
	}
	for _, p := range c.products() {
		o.Products = append(o.Products, gfx.Product{Key: p.key, Name: p.name, Percent: p.percent})
	}
	if r, ok := c.rtkInfo(); ok {
		o.RTK = &gfx.RTK{Today: r.today, Total: r.total, Rate: r.rate, Days: r.days, Letters: r.letters}
	}
	for _, s := range c.sessions() {
		state := sessionState(s)
		o.Sessions = append(o.Sessions, gfx.Session{
			Project: s.project, Detail: s.detail, Doing: s.doing,
			Context: s.context, ContextFrac: s.contextFrac, Age: s.age,
			State: state,
		})
		if state == gfx.Working {
			o.Pulse = float64(c.now.UnixMilli()%1500) / 1500
		}
	}
	for _, t := range c.tools() {
		o.Tools = append(o.Tools, gfx.Tool{Name: t.Name, Count: t.Count})
	}
	return o
}

func sessionState(s sessionInfo) gfx.SessionState {
	switch {
	case s.asking:
		return gfx.Asking
	case s.state == activity.Working:
		return gfx.Working
	case s.state == activity.Waiting:
		return gfx.Done
	}
	return gfx.Idle
}

func Snapshot(path string, demo bool) error {
	c := newCore(time.Minute, demo)
	if !demo {
		c.setReport(fetchReport(time.Minute))
		if c.report == nil {
			return c.err
		}
	}
	c.refresh()
	if !demo {
		c.rtk, _ = rtk.Load(context.Background(), c.now)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, gfx.RenderOverview(c.overview(), gfx.Options{Scale: 2}))
}
