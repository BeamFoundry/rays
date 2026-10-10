package config_test

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/internal/config"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ = Describe("Convert", func() {
	Context("Source is same as Destination", func() {
		It("can convert string to string", func() {
			v, err := Convert("42", reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
		})
	})

	Context("Source is a string", func() {
		type TextEnum struct {
			Enum[TextEnum]
			DEFAULT,
			TEXT Enum[TextEnum]
		}
		It("can convert string to time.Duration", func() {
			t := time.Minute * 30
			v, err := Convert("30m", reflect.TypeFor[time.Duration]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(t))).To(BeTrue())
		})
		It("can use a string with a TextUnmarshaleri (Enum))", func() {
			e := TextEnum{}.New().TEXT
			v, err := Convert("TEXT", reflect.TypeFor[TextEnum]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.FieldByName("Enum").Equal(reflect.ValueOf(e))).To(BeTrue())
			t, ok := v.Interface().(TextEnum)
			Expect(ok).To(BeTrue())
			Expect(t.Enum).To(Equal(e))
		})
		It("can use a string with a *TextUnmarshaler (Enum))", func() {
			e := TextEnum{}.New().TEXT
			v, err := Convert("TEXT", reflect.TypeFor[*TextEnum]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Elem().FieldByName("Enum").Equal(reflect.ValueOf(e))).To(BeTrue())
			t, ok := v.Elem().Interface().(TextEnum)
			Expect(ok).To(BeTrue())
			Expect(t.Enum).To(Equal(e))
		})
		It("will fail to convert a bad string to Enum", func() {
			_, err := Convert("BAD", reflect.TypeFor[TextEnum]())
			Expect(err).To(HaveOccurred())
		})
		It("can convert a string to []byte", func() {
			_, err := Convert("text", reflect.TypeFor[[]byte]())
			Expect(err).NotTo(HaveOccurred())
		})
		It("can convert string to bool", func() {
			v, err := Convert("true", reflect.TypeFor[bool]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(true))).To(BeTrue())
			_, err = Convert("Toto", reflect.TypeFor[bool]())
			Expect(err).To(HaveOccurred())
		})
		It("can convert string to int", func() {
			v, err := Convert("42", reflect.TypeFor[int]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(int(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[int8]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(int8(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[int16]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(int16(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[int32]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(int32(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[int64]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(int64(42)))).To(BeTrue())
		})
		It("can convert string to uint", func() {
			v, err := Convert("42", reflect.TypeFor[uint]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(uint(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[uint8]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(uint8(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[uint16]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(uint16(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[uint32]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(uint32(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[uint64]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(uint64(42)))).To(BeTrue())
		})
		It("can convert string to a float", func() {
			v, err := Convert("42", reflect.TypeFor[float32]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(float32(42)))).To(BeTrue())
			v, err = Convert("42", reflect.TypeFor[float64]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf(float64(42)))).To(BeTrue())
		})
		It("can not convert a string to []string", func() {
			_, err := Convert("text", reflect.TypeFor[[]string]())
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Destination is a string", func() {
		It("can convert an bool to string", func() {
			v, err := Convert(true, reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("true"))).To(BeTrue())
		})
		It("can convert an int to string", func() {
			v, err := Convert(int(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(int8(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(int16(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(int32(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(int64(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
		})
		It("can convert an uint to string", func() {
			v, err := Convert(uint(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(uint8(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(uint16(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(uint32(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(uint64(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
		})
		It("can convert a float to string", func() {
			v, err := Convert(float32(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
			v, err = Convert(float64(42), reflect.TypeFor[string]())
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Equal(reflect.ValueOf("42"))).To(BeTrue())
		})
	})

	Context("Source and Destination is not a string", func() {
		It("can not convert", func() {
			_, err := Convert(int(42), reflect.TypeFor[uint]())
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("ConvertNumeric", func() {
	It("can convert int to uint", func() {
		v := ConvertNumeric[uint](int(42))
		Expect(v).To(Equal(uint(42)))
	})
})

var _ = Describe("ConvertStringNumeric", func() {
	Context("Failing Test", func() {
		It("will fail convert to int", func() {
			_, err := ConvertStringNumeric[int]("true")
			Expect(err).To(HaveOccurred())
		})
		It("will fail convert to uint", func() {
			_, err := ConvertStringNumeric[uint]("true")
			Expect(err).To(HaveOccurred())
		})
		It("will fail convert to float", func() {
			_, err := ConvertStringNumeric[float32]("true")
			Expect(err).To(HaveOccurred())
		})
	})
})
