package stream_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestStream runs the stream test suite.
func TestStream(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Util/Stream Suite")
}
