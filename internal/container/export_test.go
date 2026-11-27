package container

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"time"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// SetNotifySignalContext overrides signal.NotifyContext for deterministic signal testing.
func SetNotifySignalContext(fn func(context.Context, ...os.Signal) (context.Context, context.CancelFunc)) func() {
	prev := notifySignalContext
	notifySignalContext = fn
	return func() {
		notifySignalContext = prev
	}
}

// SetShutdownTimeout overrides the runner exit wait timeout during Container.Run teardown.
func SetShutdownTimeout(d time.Duration) func() {
	prev := shutdownTimeout
	shutdownTimeout = d
	return func() {
		shutdownTimeout = prev
	}
}

// ParseScheduledTag exposes parseScheduledTag for unit testing malformed tag syntax.
func ParseScheduledTag(tag reflect.StructTag) Error {
	_, err := parseScheduledTag(tag)
	return err
}

// DefaultNotifySignalContext exposes signal.NotifyContext for test restoration.
var DefaultNotifySignalContext = signal.NotifyContext
