package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestDefaultRecordTimeAndUserTimes(t *testing.T) {
	zone := time.FixedZone("UTC-5", -5*3600)
	recordStamp := time.Date(2026, 11, 10, 8, 15, 0, 123456789, zone)
	userStamp := time.Date(2026, 11, 10, 14, 26, 33, 987654321, zone)

	expectedRecordTime := recordStamp.Format(time.RFC3339Nano)
	expectedUserTime := userStamp.Format(time.RFC3339Nano)

	tests := []struct {
		name          string
		format        string
		recordTimeKey string
		hasDuplicate  bool
	}{
		{name: "JSON", format: JSONFormat, recordTimeKey: "time", hasDuplicate: true},
		{name: "ECS", format: ECSFormat, recordTimeKey: "@timestamp", hasDuplicate: false},
		{name: "Logstash", format: LogstashFormat, recordTimeKey: "@timestamp", hasDuplicate: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			h := NewHandlerWithFormat(&out, defaultHandlerOptions(), tt.format)
			h.SetColor(false)

			derived := h.WithAttrs([]slog.Attr{slog.Time("time", userStamp)}).WithGroup("group")
			r := slog.NewRecord(recordStamp, slog.LevelInfo, "test", 0)
			r.AddAttrs(slog.Time("time", userStamp))
			if err := derived.Handle(context.Background(), r); err != nil {
				t.Fatalf("%s: handle failed: %v", tt.name, err)
			}

			// Validate JSON structure
			var js map[string]any
			if err := json.Unmarshal(out.Bytes(), &js); err != nil {
				t.Fatalf("%s: invalid json: %v\nOutput: %s", tt.name, err, out.String())
			}

			// Grouped user timestamp
			grp, ok := js["group"].(map[string]any)
			if !ok {
				t.Fatalf("%s: missing group object: %v", tt.name, js)
			}
			if grp["time"] != expectedUserTime {
				t.Errorf("%s: grouped time = %v, want %s", tt.name, grp["time"], expectedUserTime)
			}

			// Ungrouped user timestamp via map
			if js["time"] != expectedUserTime {
				t.Errorf("%s: ungrouped time = %v, want %s", tt.name, js["time"], expectedUserTime)
			}

			// Record timestamp for ECS/Logstash
			if !tt.hasDuplicate {
				if js[tt.recordTimeKey] != expectedRecordTime {
					t.Errorf("%s: record time = %v, want %s", tt.name, js[tt.recordTimeKey], expectedRecordTime)
				}
			}

			// Token-based decoding to verify duplicate key in JSON and key order
			dec := json.NewDecoder(bytes.NewReader(out.Bytes()))
			tok, err := dec.Token()
			if err != nil || tok != json.Delim('{') {
				t.Fatalf("%s: expected '{', got %v", tt.name, tok)
			}

			var topLevelTimes []string
			var foundRecordTime string
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					t.Fatalf("%s: read key error: %v", tt.name, err)
				}
				key := keyTok.(string)
				var val any
				if err := dec.Decode(&val); err != nil {
					t.Fatalf("%s: decode value for %q error: %v", tt.name, key, err)
				}
				if key == "time" {
					if s, isStr := val.(string); isStr {
						topLevelTimes = append(topLevelTimes, s)
					}
				}
				if key == "@timestamp" {
					if s, isStr := val.(string); isStr {
						foundRecordTime = s
					}
				}
			}

			if tt.hasDuplicate {
				// JSON layout has duplicate top-level "time" keys:
				// first is the record timestamp, second is the ungrouped user attribute
				if len(topLevelTimes) != 2 {
					t.Fatalf("%s: expected 2 top-level 'time' keys, got %d: %v", tt.name, len(topLevelTimes), topLevelTimes)
				}
				if topLevelTimes[0] != expectedRecordTime {
					t.Errorf("%s: record time = %q, want %q", tt.name, topLevelTimes[0], expectedRecordTime)
				}
				if topLevelTimes[1] != expectedUserTime {
					t.Errorf("%s: user time = %q, want %q", tt.name, topLevelTimes[1], expectedUserTime)
				}
			} else {
				if len(topLevelTimes) != 1 {
					t.Fatalf("%s: expected 1 top-level 'time' key, got %d: %v", tt.name, len(topLevelTimes), topLevelTimes)
				}
				if topLevelTimes[0] != expectedUserTime {
					t.Errorf("%s: user time = %q, want %q", tt.name, topLevelTimes[0], expectedUserTime)
				}
				if foundRecordTime != expectedRecordTime {
					t.Errorf("%s: record time = %q, want %q", tt.name, foundRecordTime, expectedRecordTime)
				}
			}

			// Also verify ungrouped attribute on a plain handler (without WithAttrs/WithGroup)
			out.Reset()
			plainRecord := slog.NewRecord(recordStamp, slog.LevelInfo, "plain", 0)
			plainRecord.Add(slog.Time("time", userStamp))
			plainRecord.Add(slog.Group("group", slog.Time("time", userStamp)))
			if err := h.Handle(context.Background(), plainRecord); err != nil {
				t.Fatalf("%s: plain handle failed: %v", tt.name, err)
			}
			var plainJs map[string]any
			if err := json.Unmarshal(out.Bytes(), &plainJs); err != nil {
				t.Fatalf("%s: invalid plain json: %v", tt.name, err)
			}
			if plainJs["time"] != expectedUserTime {
				t.Errorf("%s: plain ungrouped time = %v, want %s", tt.name, plainJs["time"], expectedUserTime)
			}
			plainGrp, hasGrp := plainJs["group"].(map[string]any)
			if !hasGrp || plainGrp["time"] != expectedUserTime {
				t.Errorf("%s: plain group time = %v, want %s", tt.name, plainGrp["time"], expectedUserTime)
			}
		})
	}

	// Custom ReplaceAttr callback test
	var out bytes.Buffer
	r := slog.NewRecord(recordStamp, slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Time("time", userStamp))
	h := NewHandlerWithFormat(&out, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
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
