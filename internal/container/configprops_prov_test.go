package container_test

import (
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type BeamWithConfigProperties struct {
	ConfigurationProperties `Name:"test.cfg"`
	//
	beam BeamWithValue
	fail bool
}

type BeamWithConfig struct {
	Component
	config *BeamWithConfigProperties `@:"Inject"`
}

var _ = Describe("Config", func() {
	config := map[string]any{
		"test": map[string]any{
			"cfg": map[string]any{
				"beam": map[string]any{
					"value": "config",
				},
			},
		},
	}
	m := map[string]any{
		"title":    "Les Miserables",
		"author":   "Victor Hugo",
		"pages":    2783,
		"released": true,
	}
	It("can insert config", func() {
		Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
		AddBeamFor(&BeamWithConfigProperties{})
		AddBeamFor(&BeamWithConfig{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		value, ok := GetContainer().ConfigValue("test.cfg.beam.value")
		Expect(ok).To(BeTrue())
		Expect(value).To(Equal("config"))

		instance := GetBeamDefaultInstance[*BeamWithConfig]()
		Expect(instance.config.beam.GetValue()).To(Equal("config"))
	})
	It("can override a value", func() {
		Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
		Expect(GetContainer().AddOverride("test.cfg.beam.value", "test")).NotTo(HaveOccurred())
		AddBeamFor(&BeamWithConfigProperties{})
		AddBeamFor(&BeamWithConfig{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		value, ok := GetContainer().ConfigValue("test.cfg.beam.value")
		Expect(ok).To(BeTrue())
		Expect(value).To(Equal("test"))

		instance := GetBeamDefaultInstance[*BeamWithConfig]()
		Expect(instance.config.beam.GetValue()).To(Equal("test"))
	})
	It("can refuse to inject if already set", func() {
		Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
		AddBeamFor(&BeamWithConfigProperties{})
		bcp := &BeamWithConfigProperties{}
		bcp.beam.SetValue("test")
		AddBeamFor(&BeamWithConfig{config: bcp})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		value, ok := GetContainer().ConfigValue("test.cfg.beam.value")
		Expect(ok).To(BeTrue())
		Expect(value).To(Equal("config"))

		instance := GetBeamDefaultInstance[*BeamWithConfig]()
		Expect(instance.config.beam.GetValue()).To(Equal("test"))
	})

	It("can load config at", func() {
		Expect(GetContainer().LoadConfigAt(m, "favorites.book")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		value, ok := GetContainer().ConfigValue("favorites.book.author")
		Expect(ok).To(BeTrue())
		Expect(value).To(Equal("Victor Hugo"))
	})

	It("can log error if load config after Initialize", func() {
		config := map[string]any{
			"test": map[string]any{
				"props": map[string]any{
					"value": "config",
				},
			},
		}
		Expect(GetContainer().LoadDefaultConfig(config)).NotTo(HaveOccurred())
		Expect(GetContainer().LoadDefaultConfigAt(config, "test.favorites.book")).NotTo(HaveOccurred())
		Expect(GetContainer().LoadConfigForProfile(config, "test")).NotTo(HaveOccurred())
		Expect(GetContainer().LoadConfigForProfileAt(config, "test", "test.favorites.book")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().LoadConfig(config)).To(HaveOccurred())
		Expect(GetContainer().LoadConfigAt(config, "favorites.book")).To(HaveOccurred())
		Expect(GetContainer().LoadConfigForProfile(config, "test")).To(HaveOccurred())
		Expect(GetContainer().LoadConfigForProfileAt(config, "test", "test.favorites.book")).To(HaveOccurred())
		Expect(GetContainer().LoadDefaultConfig(config)).To(HaveOccurred())
		Expect(GetContainer().LoadDefaultConfigAt(config, "test.favorites.book")).To(HaveOccurred())
	})

	It("can return an empty value", func() {
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().ConfigValue("favorites.book.author")).To(Equal(""))
	})
	It("will not fail if config does not exist", func() {
		AddBeamFor(&BeamWithConfigProperties{})
		AddBeamFor(&BeamWithConfig{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})
	It("will fail if unmarshall fails", func() {
		cfg := map[string]any{
			"test": map[string]any{
				"cfg": map[string]any{
					"fail": "fail",
				},
			},
		}
		Expect(GetContainer().LoadConfig(cfg)).NotTo(HaveOccurred())
		AddBeamFor(&BeamWithConfigProperties{})
		AddBeamFor(&BeamWithConfig{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithConfig]()
		Expect(instance.config.fail).To(BeFalse())
	})
})
