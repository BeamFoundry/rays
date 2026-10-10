package lang_test

import (
	"errors"
	"fmt"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ = Describe("Error", func() {
	name := "TestError"
	message := "An error occuped"
	It("can be created", func() {
		err := NewError(name, message)
		Expect(err).To(HaveOccurred())
		err.AddAttribute("foo", "bar")
		err.AddAttributes("ping", "pong")
		Expect(err.Name()).To(Equal(name))
		Expect(err.Message()).To(Equal(message))
		Expect(err.Error()).To(Equal(fmt.Sprintf("%s: %s", name, message)))
		err.LogError()
		err.LogErrorWithStackTrace()
		err.Log(slog.LevelDebug)
		err.LogWithStackTrace(slog.LevelDebug)
		//nolint:errcheck // Not testing it
		_, _ = err.MarshalJSON()
		Expect(err.GomegaString()).NotTo(BeEmpty())
	})
	It("can be created", func() {
		err := NewPatternSyntaxError(message)
		Expect(err).To(HaveOccurred())
	})
	It("can be created", func() {
		err := NewIndexOutOfBoundsError(message)
		Expect(err).To(HaveOccurred())
	})
	It("can be created", func() {
		err := NewErrorFrom(errors.New("Test error"))
		Expect(err.Error()).NotTo(BeEmpty())
		Expect(err).To(HaveOccurred())
		Expect(errors.Unwrap(err)).To(HaveOccurred())
	})
	It("can not be created from nil", func() {
		err := NewErrorFrom(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(IsNilError(err)).To(BeTrue())
	})
	It("can PrintStacktrace", func() {
		_, ex := String("test").CharAt(5)
		Expect(ex).To(HaveOccurred())
		GinkgoWriter.Printf("%+v", ex)
		GinkgoWriter.Printf("%v", ex)
		GinkgoWriter.Printf("%s", ex)
	})
})

var _ = Describe("WrappedError", func() {
	It("can be created", func() {
		Expect(IsNilError(NewWrappedError("TestError", nil))).To(BeTrue())
		err := NewWrappedError("TestError", errors.New("TestError"))
		Expect(IsNilError(err)).NotTo(BeTrue())
	})
})

var _ = Describe("NilError", func() {
	It("can be created", func() {
		err := &NilError{}
		Expect(err).To(HaveOccurred())
		err.Name()
		err.Message()
		_ = err.Error()
		err.GomegaString()
		err.AddAttribute("foo", "bar")
		err.LogError()
		//nolint:errcheck // Not testing it
		_, _ = err.MarshalJSON()
		err.LogErrorWithStackTrace()
		err.Log(logger.LevelInfo, "foo", "bar")
		err.LogWithStackTrace(logger.LevelDebug, "foo", "bar")
	})
})
