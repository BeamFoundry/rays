package lang_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/BeamFoundry/rays/pkg/testing"
)

func TestLang(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Lang Suite")
}
