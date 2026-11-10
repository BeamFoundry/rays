package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestDefaultRecordTimeAndUserTimes(t *testing.T) {
	stamp := time.Date(2026, 11, 5, 12, 30, 21, 123000000, time.UTC)
	var out bytes.Buffer
	h := NewHandlerWithFormat(&out, defaultHandlerOptions(), `%time(%v)|%attr(%k=%v;)%grp(%g:%attr(%k=%v;))`)
	h.SetColor(false)
	derived := h.WithAttrs([]slog.Attr{slog.Time("time", stamp)}).WithGroup("group")
	r := slog.NewRecord(stamp, slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Time("time", stamp))
	if err := derived.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "2026/11/05 12:30:21.123|") {
		t.Fatalf("record timestamp: %s", got)
	}
	if strings.Count(got, "2026/11/05") != 1 || strings.Count(got, stamp.String()) != 2 {
		t.Fatalf("user timestamps: %s", got)
	}
	out.Reset()
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "2026/11/05") != 1 || !strings.Contains(out.String(), stamp.String()) {
		t.Fatalf("ungrouped record attribute: %s", out.String())
	}
	out.Reset()
	h = NewHandlerWithFormat(&out, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			if a.Value.Kind() != slog.KindTime {
				t.Fatal("callback did not receive time")
			}
			return slog.String(a.Key, "custom")
		}
		return a
	}}, `%time(%v)|%attr(%v)`)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if out.String() != "custom|custom\n" {
		t.Fatalf("custom callback: %q", out.String())
	}
}

type countingTerminal struct {
	bytes.Buffer
	terminal bool
	calls    int
}

func (w *countingTerminal) IsTerminal() bool { w.calls++; return w.terminal }

func TestAutoColorCacheRefresh(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	t.Setenv("COLORTERM", "")
	w := &countingTerminal{terminal: true}
	h := NewHandlerWithFormat(w, nil, `%msg(%v)`)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	for range 3 {
		if err := h.Handle(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	if w.calls != 1 {
		t.Fatalf("detection called %d times", w.calls)
	}
	t.Setenv("NO_COLOR", "1")
	if h.IsColor() {
		t.Fatal("environment change ignored")
	}
	t.Setenv("NO_COLOR", "")
	if !h.IsColor() || w.calls != 2 {
		t.Fatal("environment reset ignored")
	}
	w.terminal = false
	h.EnableColor()
	if h.IsColor() || w.calls != 3 {
		t.Fatal("explicit refresh ignored")
	}
	w.terminal = true
	h.cfgMu.Lock()
	h.colorChecked = time.Now().Add(-2 * time.Second)
	h.cfgMu.Unlock()
	if !h.IsColor() || w.calls != 4 {
		t.Fatal("periodic refresh ignored")
	}
	h.SetColor(false)
	h.EnableColor()
	if !h.IsColor() {
		t.Fatal("AUTO mode not restored")
	}
}

func TestNilOptionsRemainUsable(t *testing.T) {
	lg := Logger{}.New()
	lg.SetHandlerOptions(nil)
	for _, l := range []*Logger{lg, lg.Clone()} {
		if l.options == nil {
			t.Fatal("nil options retained")
		}
		l.AddSource()
		l.SetLogLevel(LevelDebug)
		if !l.GetHandlerOptions().AddSource {
			t.Fatal("options not applied")
		}
	}
}
