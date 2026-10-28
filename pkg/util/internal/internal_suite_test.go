package internal

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestInternal runs the internal collection test suite.
func TestInternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Util/Internal Suite")
}
