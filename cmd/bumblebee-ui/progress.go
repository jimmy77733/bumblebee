//go:build windows

package main

import (
	"fmt"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type thinProgressBar struct {
	widget.ProgressBar
}

func newThinProgressBar() *thinProgressBar {
	b := &thinProgressBar{}
	b.ExtendBaseWidget(b)
	return b
}

func (b *thinProgressBar) MinSize() fyne.Size {
	th := b.Theme()
	pad := th.Size(theme.SizeNameInnerPadding) * 2
	text := fyne.MeasureText("100%", th.Size(theme.SizeNameText), fyne.TextStyle{})
	h := (text.Height + pad) * 0.4
	if h < 10 {
		h = 10
	}
	return fyne.NewSize(text.Width+pad, h)
}

type progressValue interface {
	SetValue(float64)
}

type progressAnim struct {
	mu       sync.Mutex
	gen      int
	display  float64
	expected time.Duration
	started  time.Time
	done     bool
}

func (p *progressAnim) Start(expected time.Duration, bar progressValue, onTick func()) {
	p.mu.Lock()
	p.gen++
	gen := p.gen
	p.display = 0
	if expected <= 0 {
		expected = 8 * time.Second
	}
	p.expected = expected
	p.started = time.Now()
	p.done = false
	p.mu.Unlock()
	bar.SetValue(0)

	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for range tick.C {
			stop := false
			fyne.DoAndWait(func() {
				p.mu.Lock()
				if p.gen != gen {
					p.mu.Unlock()
					stop = true
					return
				}
				finished := p.advanceLocked()
				val := p.display
				p.mu.Unlock()
				bar.SetValue(val)
				if onTick != nil {
					onTick()
				}
				if finished {
					stop = true
				}
			})
			if stop {
				return
			}
		}
	}()
}

func (p *progressAnim) Finish() {
	p.mu.Lock()
	p.done = true
	p.mu.Unlock()
}

func (p *progressAnim) Cancel() {
	p.mu.Lock()
	p.gen++
	p.done = true
	p.mu.Unlock()
}

func (p *progressAnim) ETAText() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return ""
	}
	rem := p.expected - time.Since(p.started)
	if rem < time.Second {
		return "預計剩餘不到 1 秒"
	}
	return fmt.Sprintf("預計剩餘 %d 秒", int(math.Ceil(rem.Seconds())))
}

func (p *progressAnim) advanceLocked() bool {
	elapsed := time.Since(p.started)
	var target float64
	if p.done {
		target = 1
	} else {
		if p.expected <= 0 {
			p.expected = time.Second
		}
		if elapsed.Seconds()/p.expected.Seconds() > 0.92 {
			p.expected = time.Duration(float64(p.expected) * 1.08)
		}
		target = elapsed.Seconds() / p.expected.Seconds()
		if target > 0.92 {
			target = 0.92
		}
	}
	maxStep := 0.02
	if p.done {
		maxStep = 0.08
	}
	delta := target - p.display
	if delta > maxStep {
		delta = maxStep
	} else if delta < -maxStep {
		delta = -maxStep
	}
	p.display += delta
	if p.display > 1 {
		p.display = 1
	}
	if p.display < 0 {
		p.display = 0
	}
	return p.done && p.display >= 0.999
}
