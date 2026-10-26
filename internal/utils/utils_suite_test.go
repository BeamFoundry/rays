package utils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestUtils runs the internal utils test suite.
func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal/Utils Suite")
}
