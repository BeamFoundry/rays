package lang_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

type TestException1 struct {
	*DefaultError
}

func (this TestException1) New() *TestException1 {
	this.DefaultError = NewError("TestException1", "TestException1")
	return &this
}

type TestException2 struct {
	*DefaultError
}

func (this TestException2) New() *TestException2 {
	this.DefaultError = NewError("TestException2", "TestException2")
	return &this
}

var _ = Describe("Exception", func() {
	It("can be used instead of Error", func() {
		f := func() Exception {
			return NewError("TestError", "Test Error")
		}

		//nolint:ginkgolinter // false
		Expect(f()).To(HaveOccurred())
	})
})

var _ = Describe("Try", func() {
	It("can be basic", func() {
		err := Try(func() {
			RaiseErr(NewError("TestError", "Test Error"))
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("TestError"))
	})

	It("can switch between named Error", func() {
		err := Try(func() {
			RaiseErr(NewError("TestError", "Test Error"))
		}, CatchName("TestError", func(e Error) Error {
			return e
		}), Catch(func(e Error) Error {
			Expect(e).NotTo(HaveOccurred())
			return e
		}))
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("TestError"))
	})

	It("can switch between typed Error", func() {
		err := Try(func() {
			RaiseErr(TestException1{}.New())
		}, Catch(func(e TestException1) Error {
			Expect(e).NotTo(HaveOccurred())
			return e
		}), Catch(func(e *TestException1) Error {
			Expect(e).To(HaveOccurred())
			return e
		}))
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("TestException1"))
	})

	It("can switch between typed Error", func() {
		err := Try(func() {
			RaiseErr(TestException1{}.New())
		}, Catch(func(e *TestException2) Error {
			Expect(e).NotTo(HaveOccurred())
			return e
		}),
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("TestException1"))
	})

	It("can use 'finally'", func() {
		var final bool
		err := Try(func() {
			RaiseErr(NewError("TestError", "New test panic error"))
		}, Finally(func() {
			final = true
		}),
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("TestError"))
		Expect(final).To(BeTrue())
	})

	It("can handle panic with string", func() {
		err := Try(func() {
			panic("New panic error")
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("PanicError"))
	})

	It("can handle panic with error", func() {
		err := Try(func() {
			panic(errors.New("New panic error"))
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("PanicError"))
	})

	It("can handle panic with Error", func() {
		err := Try(func() {
			panic(NewError("TestError", "New test panic error"))
		}, Catch(func(e Error) Error {
			return e
		}))
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("TestError"))
	})

	It("can not raiseerr if nil", func() {
		err := Try(func() {
			RaiseErr(nil)
		}, Catch(func(e Error) Error {
			return e
		}))
		Expect(err).NotTo(HaveOccurred())
	})

	It("can not raise if nil", func() {
		err := Try(func() {
			RaiseVal(true, nil)
		}, Catch(func(e Error) Error {
			return e
		}))
		Expect(err).NotTo(HaveOccurred())
	})

	It("can return a nil", func() {
		err := Try(func() {
			panic(errors.New("New panic error"))
		}, Catch(func(_ Error) Error {
			return nil
		}))
		Expect(err).NotTo(HaveOccurred())
	})

	It("can use RaiseNil", func() {
		err := Try(func() {
			RaiseNilErr(errors.New("New error"))
		})
		Expect(err).To(HaveOccurred())

		err = Try(func() {
			RaiseNilErr(nil)
		})
		Expect(err).To(HaveOccurred())
	})
})
