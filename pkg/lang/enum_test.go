package lang_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

var _ = Describe("Enum", func() {
	It("can evaluate an enum", func() {
		type Days struct {
			Enum[Days]
			MONDAY,
			TUESDAY,
			WEDNESDAY,
			THURSDAY,
			FRIDAY,
			SATURDAY,
			SUNDAY Enum[Days]
		}

		type Day = Enum[Days]

		weekDays := []Day{
			Days{}.New().MONDAY,
			Days{}.New().TUESDAY,
			Days{}.New().WEDNESDAY,
			Days{}.New().THURSDAY,
			Days{}.New().FRIDAY,
			Days{}.New().SATURDAY,
			Days{}.New().SUNDAY,
		}

		workDays := map[Day]bool{
			Days{}.New().MONDAY:    true,
			Days{}.New().TUESDAY:   true,
			Days{}.New().WEDNESDAY: true,
			Days{}.New().THURSDAY:  true,
			Days{}.New().FRIDAY:    true,
		}

		for _, day := range weekDays {
			if v, ok := workDays[day]; ok {
				Expect(v).To(BeTrue())
			}
		}
		Expect(Days{}.New().FRIDAY.String()).To(Equal("FRIDAY"))
		Expect(Day{}.New().FRIDAY.String()).To(Equal("FRIDAY"))
		day := Day{}

		Expect(day.From("FRIDAY")).NotTo(HaveOccurred())
		Expect(day.String()).To(Equal("FRIDAY"))

		b, err := Days{}.New().MONDAY.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(b).To(Equal([]byte("\"MONDAY\"")))

		err = day.UnmarshalJSON([]byte("\"MONDAY"))
		Expect(err).To(HaveOccurred())
		err = day.UnmarshalJSON(b)
		Expect(err).NotTo(HaveOccurred())
		Expect(day.Ordinal()).To(Equal(0))

		t, err := Days{}.New().FRIDAY.MarshalText()
		Expect(err).NotTo(HaveOccurred())

		v := ValErr(Days{}.New().FRIDAY.Value())
		Expect(day.Scan(v)).NotTo(HaveOccurred())
		Expect(day.String()).To(Equal("FRIDAY"))

		err = day.UnmarshalText([]byte("friday"))
		Expect(err).To(HaveOccurred())
		err = day.UnmarshalText(t)
		Expect(err).NotTo(HaveOccurred())
		Expect(day.String()).To(Equal("FRIDAY"))

		var switched bool
		switch day {
		case Days{}.New().FRIDAY:
			switched = true
		default:
			switched = false
		}
		Expect(switched).To(BeTrue())

		before := Day{}.New().MONDAY.Compare(Day{}.New().FRIDAY) < 0
		Expect(before).To(BeTrue())

		d := Days{}
		err = d.UnmarshalText(t)
		Expect(err).NotTo(HaveOccurred())
		Expect(d.String()).To(Equal("FRIDAY"))
	})

	It("can use ordinale serializer", func() {
		type IsSecure struct {
			Enum[IsSecure]
			UNSECURE,
			SECURE Enum[IsSecure] `Enum:"Ordinal"`
		}

		b, err := IsSecure{}.New().SECURE.MarshalText()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(b)).To(Equal("1"))

		b, err = IsSecure{}.New().SECURE.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(b)).To(Equal("1"))

		v, err := IsSecure{}.New().SECURE.Value()
		Expect(err).NotTo(HaveOccurred())

		secure := Enum[IsSecure]{}
		err = secure.UnmarshalJSON([]byte("1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(secure.Name()).To(Equal("SECURE"))
		Expect(secure.Scan(v)).NotTo(HaveOccurred())
		Expect(secure.Scan("SECURE")).To(HaveOccurred())
	})

	It("can use a custom value", func() {
		type ProtocolEnum struct {
			LowerEnumStringer[ProtocolEnum]
			HTTP,
			HTTPS Enum[ProtocolEnum]
		}

		type Protocol = Enum[ProtocolEnum]

		Expect(Protocol{}.New().HTTP.String()).To(Equal("http"))
		Expect(Protocol{}.New().HTTPS.String()).To(Equal("https"))
	})

	It("can use a custom value", func() {
		type Protocols struct {
			TitleEnumStringer[Protocols]
			HTTP,
			HTTPS Enum[Protocols]
		}

		type Protocol = Enum[Protocols]

		Expect(Protocol{}.New().HTTP.String()).To(Equal("Http"))
		Expect(Protocol{}.New().HTTPS.String()).To(Equal("Https"))
	})
})

type EnumFormat struct {
	JSON,
	YAML Enum[EnumFormat]
}

var Formats = Format{}.New()

type Format struct {
	LowerEnumStringer[EnumFormat]
	Enum[EnumFormat]
}

func (this Format) Convert(format Enum[EnumFormat]) string {
	return this.ToString(format)
}

var _ = Describe("New Format", func() {
	It("can use a valuer", func() {
		Expect(Formats.JSON.String()).To(Equal("JSON"))
		Expect(Format{}.Convert(Formats.JSON)).To(Equal("json"))
	})
})
