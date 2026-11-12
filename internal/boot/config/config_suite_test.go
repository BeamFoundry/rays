package config_test

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BeamFoundry/rays/internal/boot/config"
	"github.com/BeamFoundry/rays/internal/boot/logger"
	_ "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/BeamFoundry/rays/pkg/util"
)

type failingFS struct{}

func (failingFS) Open(_ string) (fs.File, error) {
	return nil, errors.New("failing filesystem error")
}

type unreadableFileFS struct {
	fs.FS
}

func (u unreadableFileFS) Open(name string) (fs.File, error) {
	f, err := u.FS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	stat, err := f.Stat()
	if err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, fmt.Errorf("stat %s: %w", name, err)
	}
	if stat.IsDir() {
		return f, nil
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("close %s: %w", name, err)
	}
	return nil, errors.New("cannot read file")
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal/Config Suite")
}

/*
var _ = AfterEach(func() {
	config.NewConfig().ResetDefaults()
})
*/

type Book struct {
	title    *string
	author   string
	pages    int `validate:"gt=0"`
	released bool
}

type BookElemsString struct {
	books []string
}

type BookElem struct {
	book Book
}

type BookElemsSlice struct {
	books []Book `validate:"dive"`
}

type BookElemsMapObject struct {
	books map[string]Book
}

type BookElemsMapObjectPointer struct {
	books map[string]*Book
}

type BookElemsMapStringPointer struct {
	books map[string]*string
}

type BookElemsMap struct {
	books []map[string]string
}

type BookElemsMapEntry struct {
	books []MapEntry[string, string]
}

type BookElemsMapEntryPointer struct {
	books []MapEntry[string, *Book]
}

type BookElemPointer struct {
	book *Book
}

type TranslationMap struct {
	en map[string]string
	fr map[string]string
}

//go:embed testdata/*
var testdata embed.FS

var _ = Describe("config", func() {
	lesmis := map[string]any{
		"title":    "Les Miserables",
		"author":   "Victor Hugo",
		"pages":    2783,
		"released": true,
	}

	germinal := map[string]any{
		"title":    "Germinal",
		"author":   "Emile Zola",
		"pages":    562,
		"released": true,
	}

	favoritesTitles := map[string]any{
		"favorites": map[string]any{
			"books": []any{
				"Les Miserables",
				"Germinal",
			},
		},
	}

	favorite := map[string]any{
		"favorite": map[string]any{
			"book": lesmis,
		},
	}

	favorites := map[string]any{
		"favorites": map[string]any{
			"books": map[string]any{
				"Les Miserables": lesmis,
				"Germinal":       germinal,
			},
		},
	}

	favoritesList := map[string]any{
		"favorites": map[string]any{
			"books": []any{
				lesmis,
				germinal,
			},
		},
	}

	favoritesListTitles := map[string]any{
		"favorites": map[string]any{
			"books": map[string]any{
				"Les Miserables": "Victor Hugo",
				"Germinal":       "Emile Zola",
				"Nil":            nil,
			},
		},
	}

	favoritesListElems := map[string]any{
		"favorites": map[string]any{
			"books": []any{
				map[string]any{"Les Miserables": "Victor Hugo"},
				map[string]any{"Germinal": "Emile Zola"},
			},
		},
	}

	favoritesListElemsTitles := map[string]any{
		"favorites": map[string]any{
			"books": []any{
				map[string]any{"Les Miserables": "Victor Hugo"},
				map[string]any{"Germinal": "Emile Zola"},
				map[string]any{"Nil": nil},
			},
		},
	}

	translations := map[string]any{
		"translations": map[string]any{
			"en": map[string]any{
				"title": "Title",
			},
			"fr": map[string]any{
				"title": "Titre",
			},
		},
	}

	It("can load an object", func() {
		book := &Book{}
		c := config.NewConfigFor(lesmis)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.InitConfig()).To(HaveOccurred())
		err := c.Unmarshal(Book{})
		Expect(err).To(HaveOccurred())

		Expect(c.Unmarshal(book)).NotTo(HaveOccurred())
		Expect(book.title).NotTo(BeNil())
		Expect(book.author).To(Equal("Victor Hugo"))
		Expect(book.pages).To(Equal(2783))
		Expect(book.released).To(BeTrue())
	})

	It("can load an object, using pointer", func() {
		book := &Book{}
		c := config.NewConfigFor(lesmis)
		Expect(c.LoadConfig(map[string]any{"title": nil})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.LoadConfig(map[string]any{"title": nil})).To(HaveOccurred())
		err := c.Unmarshal(Book{})
		Expect(err).To(HaveOccurred())

		Expect(c.Unmarshal(book)).NotTo(HaveOccurred())
		Expect(book.title).To(BeNil())
		Expect(book.author).To(Equal("Victor Hugo"))
		Expect(book.pages).To(Equal(2783))
		Expect(book.released).To(BeTrue())
	})

	It("can load an object from default", func() {
		book := &Book{}
		c := config.NewConfig()
		Expect(c.LoadDefaultConfig(lesmis)).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.LoadDefaultConfig(lesmis)).To(HaveOccurred())
		err := c.Unmarshal(Book{})
		Expect(err).To(HaveOccurred())

		Expect(c.Unmarshal(book)).NotTo(HaveOccurred())
		Expect(book.author).To(Equal("Victor Hugo"))
		Expect(book.pages).To(Equal(2783))
		Expect(book.released).To(BeTrue())
	})

	It("can load an object containing slice", func() {
		elems := &BookElemsString{}
		c := config.NewConfigFor(favoritesTitles)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.books[0]).To(Equal("Les Miserables"))
		Expect(elems.books[1]).To(Equal("Germinal"))
	})

	It("can load an object containing other objects", func() {
		elem := &BookElem{}
		c := config.NewConfigFor(favorite)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorite")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elem)).NotTo(HaveOccurred())
		Expect(elem.book.author).To(Equal("Victor Hugo"))
		Expect(elem.book.pages).To(Equal(2783))
		Expect(elem.book.released).To(BeTrue())
	})

	It("can load an object at a path, with profile", func() {
		elem := &BookElem{}
		c := config.NewConfig()
		Expect(c.LoadConfigForProfile(favoritesTitles, "test")).NotTo(HaveOccurred())
		Expect(c.LoadConfigForProfileAt(lesmis, "test", "favorite.books.lesmis.book")).NotTo(HaveOccurred())
		Expect(c.LoadConfigForProfileAt(germinal, "test", "favorite.books.germinal.book")).NotTo(HaveOccurred())
		Expect(c.SetActiveProfiles([]string{"test"})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.SetActiveProfiles([]string{"test"})).To(HaveOccurred())
		Expect(c.LoadConfigForProfile(favoritesTitles, "test")).To(HaveOccurred())
		Expect(c.LoadConfigForProfileAt(lesmis, "test", "favorite.books.lesmis.book")).To(HaveOccurred())
		subkey, err := c.GetSubConfig("favorite.books.lesmis")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elem)).NotTo(HaveOccurred())
		Expect(elem.book.author).To(Equal("Victor Hugo"))
		Expect(elem.book.pages).To(Equal(2783))
		Expect(elem.book.released).To(BeTrue())
	})

	It("can load an object containing other objects using slice", func() {
		elems := &BookElemsSlice{}
		c := config.NewConfigFor(favoritesList)
		Expect(c.AddOverride("favorites.books.[1].released", "false")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(*elems.books[0].title).To(Equal("Les Miserables"))
		Expect(elems.books[1].released).To(BeFalse())
	})

	It("can load an object containing other objects using map", func() {
		elems := &BookElemsMapObject{}
		c := config.NewConfigFor(favorites)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(*elems.books["Les Miserables"].title).To(Equal("Les Miserables"))
	})

	It("can load an object containing other objects pointer using map", func() {
		elems := &BookElemsMapObjectPointer{}
		c := config.NewConfigFor(favorites)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(*elems.books["Les Miserables"].title).To(Equal("Les Miserables"))
	})
	It("can load an object using maps of maps", func() {
		type Favorites struct {
			favorites map[string]map[string]Book
		}
		elems := &Favorites{}
		c := config.NewConfigFor(favorites)
		Expect(c.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.favorites["books"]).To(HaveKey("Les Miserables"))
	})
	It("can load an object using slice of maps", func() {
		elems := &BookElemsMap{}
		c := config.NewConfigFor(favoritesListElems)
		Expect(c.AddOverride("favorites.books.[1].germinal", "emile zola")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.books[0]["Les Miserables"]).To(Equal("Victor Hugo"))
		Expect(elems.books[1]["Germinal"]).To(Equal("emile zola"))
	})
	It("can load an object using slice of MapEntry", func() {
		elems := &BookElemsMapEntry{}
		c := config.NewConfigFor(favoritesListElems)
		Expect(c.AddOverride("favorites.books.[1].value", "emile zola")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.books[0].Key()).To(Equal("Les Miserables"))
		Expect(elems.books[0].Value()).To(Equal("Victor Hugo"))
		Expect(elems.books[1].Value()).To(Equal("emile zola"))
	})
	It("can load an object using slice of MapEntry with struct pointers", func() {
		elems := &BookElemsMapEntryPointer{}
		c := config.NewConfigFor(map[string]any{
			"favorites": map[string]any{
				"books": []any{
					map[string]any{"Les Miserables": lesmis},
					map[string]any{"Germinal": germinal},
				},
			},
		})
		Expect(c.AddOverride("favorites.books.[0].value.pages", "3000")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.books).To(HaveLen(2))
		Expect(elems.books[0].Key()).To(Equal("Les Miserables"))
		Expect(elems.books[0].Value()).NotTo(BeNil())
		Expect(elems.books[0].Value().author).To(Equal("Victor Hugo"))
		Expect(elems.books[0].Value().pages).To(Equal(3000))
		Expect(*elems.books[0].Value().title).To(Equal("Les Miserables"))
		Expect(elems.books[1].Key()).To(Equal("Germinal"))
		Expect(elems.books[1].Value()).NotTo(BeNil())
		Expect(elems.books[1].Value().author).To(Equal("Emile Zola"))
	})
	It("can handle nil struct pointer in MapEntry", func() {
		elems := &BookElemsMapEntryPointer{}
		c := config.NewConfigFor(map[string]any{
			"favorites": map[string]any{
				"books": []any{
					map[string]any{"Les Miserables": nil},
				},
			},
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.books).To(HaveLen(1))
		Expect(elems.books[0].Key()).To(Equal("Les Miserables"))
		Expect(elems.books[0].Value()).To(BeNil())
	})
	It("fails on invalid struct section in MapEntry pointer", func() {
		elems := &BookElemsMapEntryPointer{}
		c := config.NewConfigFor(map[string]any{
			"favorites": map[string]any{
				"books": []any{
					map[string]any{"Les Miserables": "not a struct"},
				},
			},
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).To(HaveOccurred())
	})
	It("can load an object containing a pointer-to-struct field with nested section", func() {
		elem := &BookElemPointer{}
		c := config.NewConfigFor(map[string]any{
			"book": lesmis,
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.Unmarshal(elem)).NotTo(HaveOccurred())
		Expect(elem.book).NotTo(BeNil())
		Expect(elem.book.author).To(Equal("Victor Hugo"))
		Expect(elem.book.pages).To(Equal(2783))
	})
	It("can handle nil pointer-to-struct field in an object", func() {
		elem := &BookElemPointer{}
		c := config.NewConfigFor(map[string]any{
			"book": nil,
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.Unmarshal(elem)).NotTo(HaveOccurred())
		Expect(elem.book).To(BeNil())
	})
	It("fails when pointer-to-struct field has invalid type instead of nested section", func() {
		elem := &BookElemPointer{}
		c := config.NewConfigFor(map[string]any{
			"book": "invalid-not-a-map",
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.Unmarshal(elem)).To(HaveOccurred())
	})
	It("fails when nested pointer-to-struct unmarshaling fails", func() {
		elem := &BookElemPointer{}
		c := config.NewConfigFor(map[string]any{
			"book": map[string]any{
				"pages": "not-an-int",
			},
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.Unmarshal(elem)).To(HaveOccurred())
	})
	It("can handle nil struct pointer in map", func() {
		elems := &BookElemsMapObjectPointer{}
		c := config.NewConfigFor(map[string]any{
			"favorites": map[string]any{
				"books": map[string]any{
					"Nil": nil,
				},
			},
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())
		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.books["Nil"]).To(BeNil())
	})
	It("fails when struct pointer in map has invalid section", func() {
		elems := &BookElemsMapObjectPointer{}
		c := config.NewConfigFor(map[string]any{
			"favorites": map[string]any{
				"books": map[string]any{
					"Bad": "not-a-map",
				},
			},
		})
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())
		Expect(subkey.Unmarshal(elems)).To(HaveOccurred())
	})
	It("can duplicate default config", func() {
		elems := &BookElemsMapObject{}
		c := config.NewConfig()
		Expect(c.LoadDefaultConfig(favorites)).NotTo(HaveOccurred())
		Expect(c.CopyDefaultConfig("favorites.books", "books")).NotTo(HaveOccurred())
		Expect(c.CopyDefaultConfigForProfile("favorites.books", "test.books", "test")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.CopyDefaultConfig("favorites.books", "books")).To(HaveOccurred())
		Expect(c.CopyDefaultConfigForProfile("favorites.books", "test.books", "test")).To(HaveOccurred())
		Expect(c.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(*elems.books["Les Miserables"].title).To(Equal("Les Miserables"))
	})

	It("will not fail on nil slice/map", func() {
		nilElem := map[string]any{
			"favorites": map[string]any{
				"books": nil,
			},
		}
		c := config.NewConfigFor(nilElem)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(&BookElemsSlice{})).NotTo(HaveOccurred())
		Expect(subkey.Unmarshal(&BookElemsMapObject{})).NotTo(HaveOccurred())
	})

	It("can load an object containing map to string pointers", func() {
		elems := &BookElemsMapStringPointer{}
		c := config.NewConfigFor(favoritesListTitles)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
		Expect(elems.books["Les Miserables"]).ToNot(BeNil())
		Expect(*elems.books["Les Miserables"]).To(Equal("Victor Hugo"))
	})

	It("can get sub keys", func() {
		book := &Book{}
		c := config.NewConfigFor(favorite)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorite.book")
		Expect(err).NotTo(HaveOccurred())
		Expect(subkey.Unmarshal(book)).NotTo(HaveOccurred())
		Expect(book.author).To(Equal("Victor Hugo"))
		Expect(book.pages).To(Equal(2783))
		Expect(book.released).To(BeTrue())
	})

	It("can load config at a level", func() {
		c := config.NewConfig()
		Expect(c.LoadConfigAt(lesmis, "favorites.book")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.LoadConfigAt(lesmis, "favorites.book")).To(HaveOccurred())
		book := &Book{}
		subkey, err := c.GetSubConfig("favorites.book")
		Expect(err).NotTo(HaveOccurred())
		Expect(subkey.Unmarshal(book)).NotTo(HaveOccurred())
		Expect(book.author).To(Equal("Victor Hugo"))
		Expect(book.pages).To(Equal(2783))
		Expect(book.released).To(BeTrue())
	})

	It("can load default config at a level", func() {
		c := config.NewConfig()
		Expect(c.LoadDefaultConfigAt(lesmis, "favorites.book")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.LoadDefaultConfigAt(lesmis, "favorites.book")).To(HaveOccurred())
		book := &Book{}
		subkey, err := c.GetSubConfig("favorites.book")
		Expect(err).NotTo(HaveOccurred())
		Expect(subkey.Unmarshal(book)).NotTo(HaveOccurred())
		Expect(book.author).To(Equal("Victor Hugo"))
		Expect(book.pages).To(Equal(2783))
		Expect(book.released).To(BeTrue())
	})

	It("can use overrides", func() {
		book := &Book{}
		c := config.NewConfigFor(favorite)
		Expect(c.AddOverride("favorite.book.author", "victor hugo")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.AddOverride("favorite.book.author", "victor hugo")).To(HaveOccurred())
		Expect(c.GetConfigValue("favorite.book.author")).To(Equal("victor hugo"))
		subkey, err := c.GetSubConfig("favorite.book")
		Expect(err).NotTo(HaveOccurred())
		Expect(subkey.Unmarshal(book)).NotTo(HaveOccurred())
		Expect(book.author).To(Equal("victor hugo"))
	})

	It("can load from FS", func() {
		c := config.NewConfig()
		Expect(c.AddConfigFS(testdata)).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.AddConfigFS(testdata)).To(HaveOccurred())

		elem := &BookElem{}
		subkey, err := c.GetSubConfig("favorite")
		Expect(err).NotTo(HaveOccurred())
		err = subkey.Unmarshal(BookElem{})
		Expect(err).To(HaveOccurred())

		Expect(subkey.Unmarshal(elem)).NotTo(HaveOccurred())
		Expect(elem.book.author).To(Equal("Victor Hugo"))
	})

	It("can load from FS to default", func() {
		c := config.NewConfig()
		Expect(c.AddDefaultConfigFS(testdata)).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		Expect(c.AddDefaultConfigFS(testdata)).To(HaveOccurred())

		elem := &BookElem{}
		subkey, err := c.GetSubConfig("favorite")
		Expect(err).NotTo(HaveOccurred())
		err = subkey.Unmarshal(BookElem{})
		Expect(err).To(HaveOccurred())

		Expect(subkey.Unmarshal(elem)).NotTo(HaveOccurred())
		Expect(elem.book.author).To(Equal("Victor Hugo"))
		c.Reset()
	})

	It("can load from FS with profile", func() {
		c := config.NewConfig()
		Expect(c.AddConfigFS(testdata)).NotTo(HaveOccurred())
		Expect(c.SetActiveProfiles([]string{"test"})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())

		Expect(c.ActiveProfiles()).To(Equal([]string{"test"}))

		elem := &BookElem{}
		subkey, err := c.GetSubConfig("favorite")
		Expect(err).NotTo(HaveOccurred())
		err = subkey.Unmarshal(BookElem{})
		Expect(err).To(HaveOccurred())

		Expect(subkey.Unmarshal(elem)).NotTo(HaveOccurred())
		Expect(elem.book.author).To(Equal("Emile Zola"))
	})

	It("can get a value", func() {
		c := config.NewConfigFor(lesmis)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		author, err := c.GetConfigValue("author")
		Expect(err).NotTo(HaveOccurred())
		Expect(author).To(Equal("Victor Hugo"))
		c.Reset()
		Expect(c.LoadConfig(favorite)).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		author, err = c.GetConfigValue("favorite.book.author")
		Expect(err).NotTo(HaveOccurred())
		Expect(author).To(Equal("Victor Hugo"))
	})
	It("can load map keys", func() {
		c := config.NewConfigFor(translations)
		Expect(c.InitConfig()).NotTo(HaveOccurred())

		en, err := c.GetConfigValue("translations.en.title")
		Expect(err).NotTo(HaveOccurred())
		Expect(en).NotTo(BeEmpty())
		Expect(en).To(Equal("Title"))

		fr, err := c.GetConfigValue("translations.fr.title")
		Expect(err).NotTo(HaveOccurred())
		Expect(fr).NotTo(BeEmpty())
		Expect(fr).To(Equal("Titre"))

		trans := &TranslationMap{}
		subkey, err := c.GetSubConfig("translations")
		Expect(err).NotTo(HaveOccurred())
		err = subkey.Unmarshal(trans)
		Expect(err).NotTo(HaveOccurred())

		Expect(trans.en).NotTo(BeEmpty(), "EN")
		Expect(trans.fr).NotTo(BeEmpty(), "FR")
	})

	It("can return an error if a key does not exists", func() {
		c := config.NewConfigFor(lesmis)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		_, err := c.GetConfigValue("book")
		Expect(err).To(HaveOccurred())
		Expect(c.Unmarshal(Book{})).To(HaveOccurred())
		c.Reset()
		Expect(c.LoadConfig(favorite)).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		_, err = c.GetConfigValue("favorite.books")
		Expect(err).To(HaveOccurred())
		_, err = c.GetSubConfig("favorite.books")
		Expect(err).To(HaveOccurred())
		_, err = c.GetSubConfig("favorite.book.title")
		Expect(err).To(HaveOccurred())
		_, err = c.GetSubConfig("favorites")
		Expect(err).To(HaveOccurred())
		_, err = c.GetSubConfig("favorite.book")
		Expect(err).NotTo(HaveOccurred())
	})

	It("can error on validation", func() {
		book := &Book{}
		c := config.NewConfigFor(lesmis)
		Expect(c.LoadConfig(map[string]any{"pages": 0})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		err := c.Unmarshal(book)
		Expect(err).To(HaveOccurred())
	})

	It("can error on deep validation", func() {
		favs := map[string]any{
			"favorites": map[string]any{
				"books": []any{
					lesmis,
					map[string]any{
						"title":    "Germinal",
						"author":   "Emile Zola",
						"pages":    0,
						"released": true,
					},
				},
			},
		}

		elems := &BookElemsSlice{}
		c := config.NewConfigFor(favs)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).To(HaveOccurred())
	})

	Context("It will error if key conversion fail", func() {
		It("on direct key", func() {
			c := config.NewConfigFor(lesmis)
			Expect(c.LoadConfigAt(map[string]any{"released": "released"}, "favorite.book")).NotTo(HaveOccurred())
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			book := &Book{}
			subkey, err := c.GetSubConfig("favorite.book")
			Expect(err).NotTo(HaveOccurred())
			err = subkey.Unmarshal(book)
			Expect(err).To(HaveOccurred())
		})
		It("On struct", func() {
			c := config.NewConfigFor(favorite)
			Expect(c.LoadConfigAt(map[string]any{"released": "released"}, "favorite.book")).NotTo(HaveOccurred())
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			book := &BookElem{}
			subkey, err := c.GetSubConfig("favorite")
			Expect(err).NotTo(HaveOccurred())
			err = subkey.Unmarshal(book)
			Expect(err).To(HaveOccurred())
		})
		It("On slice of struct", func() {
			elems := &BookElemsSlice{}
			c := config.NewConfigFor(favoritesList)
			Expect(c.AddOverride("favorites.books.[1].released", "release")).NotTo(HaveOccurred())
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			subkey, err := c.GetSubConfig("favorites")
			Expect(err).NotTo(HaveOccurred())
			Expect(subkey.Unmarshal(elems)).To(HaveOccurred())
		})
		It("On slice of maps", func() {
			elems := &BookElemsMap{}
			c := config.NewConfigFor(favoritesListElemsTitles)
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			subkey, err := c.GetSubConfig("favorites")
			Expect(err).NotTo(HaveOccurred())
			Expect(subkey.Unmarshal(elems)).To(HaveOccurred())
		})
		It("On slice of element", func() {
			c := config.NewConfigFor(map[string]any{
				"data": map[string]any{
					"bools": []any{
						"true",
						"any",
					},
				},
			})
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			data := struct {
				bools []bool
			}{
				bools: []bool{},
			}
			subkey, err := c.GetSubConfig("data")
			Expect(err).NotTo(HaveOccurred())
			Expect(subkey.Unmarshal(&data)).To(HaveOccurred())
		})
		It("On map of struct", func() {
			elems := &BookElemsMapObject{}
			c := config.NewConfigFor(favorites)
			Expect(c.LoadConfigAt(map[string]any{
				"Les Miserables": map[string]any{
					"released": "released",
				},
			}, "favorites.books")).NotTo(HaveOccurred())
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			subkey, err := c.GetSubConfig("favorites")
			Expect(err).NotTo(HaveOccurred())

			Expect(subkey.Unmarshal(elems)).To(HaveOccurred())
		})
		It("On map of struct pointer", func() {
			elems := &BookElemsMapObjectPointer{}
			c := config.NewConfigFor(favorites)
			Expect(c.LoadConfigAt(map[string]any{
				"Les Miserables": map[string]any{
					"released": "released",
				},
			}, "favorites.books")).NotTo(HaveOccurred())
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			subkey, err := c.GetSubConfig("favorites")
			Expect(err).NotTo(HaveOccurred())

			Expect(subkey.Unmarshal(elems)).To(HaveOccurred())
		})
		It("On map of element", func() {
			c := config.NewConfigFor(map[string]any{
				"data": map[string]any{
					"bools": map[string]any{
						"true": "true",
						"any":  "any",
					},
				},
			})
			Expect(c.InitConfig()).NotTo(HaveOccurred())
			data := struct {
				bools map[string]bool
			}{
				bools: map[string]bool{},
			}
			subkey, err := c.GetSubConfig("data")
			Expect(err).NotTo(HaveOccurred())
			Expect(subkey.Unmarshal(&data)).To(HaveOccurred())
		})
	})

	It("will not error if a key is not a slice as expected", func() {
		elems := &BookElemsSlice{}
		c := config.NewConfigFor(favorites)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
	})

	It("will not error if a key is not a slice of struct as expected", func() {
		elems := &BookElemsSlice{}
		c := config.NewConfigFor(favoritesTitles)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
	})

	It("will not error if a key is not a slice of maps", func() {
		elems := &BookElemsMap{}
		c := config.NewConfigFor(favoritesTitles)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		subkey, err := c.GetSubConfig("favorites")
		Expect(err).NotTo(HaveOccurred())

		Expect(subkey.Unmarshal(elems)).NotTo(HaveOccurred())
	})
	It("can error if validation fails", func() {
		c := config.NewConfigFor(lesmis)
		Expect(c.LoadConfigAt(map[string]any{"pages": 0}, "favorite.book")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		book := &Book{}
		subkey, err := c.GetSubConfig("favorite.book")
		Expect(err).NotTo(HaveOccurred())
		err = subkey.Unmarshal(book)
		Expect(err).To(HaveOccurred())
	})

	It("will fail getting an object as value", func() {
		c := config.NewConfigFor(favoritesList)
		Expect(c.AddOverride("favorites.books.1.released", "release")).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		_, err := c.GetConfigValue("favorites.books.1.released")
		Expect(err).NotTo(HaveOccurred())
		_, err = c.GetConfigValue("favorites.books")
		Expect(err).To(HaveOccurred())
	})

	It("will error loading a map as a slice", func() {
		c := config.NewConfigFor(favoritesTitles)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		_, err := c.GetSubConfig("favorites.books.Germinal")
		Expect(err).To(HaveOccurred())
		_, err = c.GetConfigValue("favorites.books.released.Germinal")
		Expect(err).To(HaveOccurred())
	})

	It("can set a custom logger", func() {
		c := config.NewConfig()
		initialLogger := config.GetConfigLogger(c)
		Expect(initialLogger).NotTo(BeNil())

		c.SetLogger(nil)
		Expect(config.GetConfigLogger(c)).To(BeIdenticalTo(initialLogger))

		customLogger := logger.NewLogger()
		c.SetLogger(customLogger)
		Expect(config.GetConfigLogger(c)).To(BeIdenticalTo(customLogger))
	})

	It("can clear result and allow reinitialization", func() {
		c := config.NewConfigFor(favorite)
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		v, err := c.GetConfigValue("favorite.book.author")
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal("Victor Hugo"))

		c.Clear()
		Expect(c.InitConfig()).NotTo(HaveOccurred())
		v, err = c.GetConfigValue("favorite.book.author")
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal("Victor Hugo"))
	})

	It("reuses parent logger in GetSubConfig and creates independent logger in NewConfig", func() {
		c1 := config.NewConfig()
		c2 := config.NewConfig()
		Expect(config.GetConfigLogger(c1)).NotTo(BeNil())
		Expect(config.GetConfigLogger(c2)).NotTo(BeNil())
		Expect(config.GetConfigLogger(c1)).NotTo(BeIdenticalTo(config.GetConfigLogger(c2)))

		customLogger := logger.NewLogger()
		c1.SetLogger(customLogger)
		Expect(config.GetConfigLogger(c1)).To(BeIdenticalTo(customLogger))

		Expect(c1.LoadConfig(favorite)).NotTo(HaveOccurred())
		Expect(c1.InitConfig()).NotTo(HaveOccurred())

		sub, err := c1.GetSubConfig("favorite")
		Expect(err).NotTo(HaveOccurred())
		Expect(config.GetConfigLogger(sub)).To(BeIdenticalTo(customLogger))
	})

	It("ensures subconfiguration uses independent copies and Clear does not mutate parent or prevent reinitialization", func() {
		parent := config.NewConfigFor(favorite)
		Expect(parent.InitConfig()).NotTo(HaveOccurred())

		sub, err := parent.GetSubConfig("favorite")
		Expect(err).NotTo(HaveOccurred())

		val, err := sub.GetConfigValue("book.author")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("Victor Hugo"))

		// Clearing subconfiguration should not mutate the parent's nested map
		sub.Clear()
		parentVal, err := parent.GetConfigValue("favorite.book.author")
		Expect(err).NotTo(HaveOccurred())
		Expect(parentVal).To(Equal("Victor Hugo"))

		// Clearing subconfiguration should not prevent reinitialization
		Expect(sub.InitConfig()).NotTo(HaveOccurred())
		reinitVal, err := sub.GetConfigValue("book.author")
		Expect(err).NotTo(HaveOccurred())
		Expect(reinitVal).To(Equal("Victor Hugo"))
	})

	It("deep-clones nested maps and slices so child LoadConfig updates cannot mutate parent configuration", func() {
		cfg := map[string]any{
			"service": map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
				"tags": []any{"primary", "v1"},
				"servers": []any{
					map[string]any{"name": "srv1"},
				},
			},
		}
		parent := config.NewConfigFor(cfg)
		Expect(parent.InitConfig()).NotTo(HaveOccurred())

		child, err := parent.GetSubConfig("service")
		Expect(err).NotTo(HaveOccurred())

		// Update child database host via LoadConfig merge
		Expect(child.LoadConfig(map[string]any{
			"database": map[string]any{
				"host": "remotehost",
			},
		})).NotTo(HaveOccurred())

		// Parent database host must remain unchanged
		pHost, err := parent.GetConfigValue("service.database.host")
		Expect(err).NotTo(HaveOccurred())
		Expect(pHost).To(Equal("localhost"))

		pPort, err := parent.GetConfigValue("service.database.port")
		Expect(err).NotTo(HaveOccurred())
		Expect(pPort).To(Equal("5432"))

		// Child reflects the updated value after initialization
		Expect(child.InitConfig()).NotTo(HaveOccurred())
		cHost, err := child.GetConfigValue("database.host")
		Expect(err).NotTo(HaveOccurred())
		Expect(cHost).To(Equal("remotehost"))

		cPort, err := child.GetConfigValue("database.port")
		Expect(err).NotTo(HaveOccurred())
		Expect(cPort).To(Equal("5432"))

		// Ensure deep clone handles nil map and nil slice
		Expect(config.DeepCloneMap(nil)).To(BeNil())
		Expect(config.DeepCloneSlice(nil)).To(BeNil())
	})

	It("handles filesystem walk errors without panic and returns error in YamlConfigFSLoader", func() {
		loader := &config.YamlConfigFSLoader{}
		called := false
		err := loader.ParseConfigurationFilesystem(failingFS{}, func(_ map[string]any, _ string) {
			called = true
		})
		Expect(err).To(HaveOccurred())
		Expect(called).To(BeFalse())

		c := config.NewConfig()
		Expect(c.AddConfigFS(failingFS{})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).To(HaveOccurred())
	})

	It("returns error for malformed YAML files and propagates through InitConfig", func() {
		malformedFS := fstest.MapFS{
			"application.yaml": &fstest.MapFile{
				Data: []byte("invalid: [yaml: foo"),
			},
		}
		loader := &config.YamlConfigFSLoader{}
		err := loader.ParseConfigurationFilesystem(malformedFS, func(_ map[string]any, _ string) {})
		Expect(err).To(HaveOccurred())

		c := config.NewConfig()
		Expect(c.AddConfigFS(malformedFS)).NotTo(HaveOccurred())
		Expect(c.InitConfig()).To(HaveOccurred())

		cDefault := config.NewConfig()
		Expect(cDefault.AddDefaultConfigFS(malformedFS)).NotTo(HaveOccurred())
		Expect(cDefault.InitConfig()).To(HaveOccurred())
	})

	It("returns error when reading configuration file fails", func() {
		loader := &config.YamlConfigFSLoader{}
		err := loader.ParseConfigurationFilesystem(unreadableFileFS{FS: testdata}, func(_ map[string]any, _ string) {})
		Expect(err).To(HaveOccurred())
	})

	It("does not mutate base configuration when merging profile configs, preserving it across Clear and reinit", func() {
		c := config.NewConfig()
		Expect(c.LoadConfig(map[string]any{
			"server": map[string]any{
				"host": "basehost",
				"port": 8080,
			},
		})).NotTo(HaveOccurred())
		Expect(c.LoadConfigForProfile(map[string]any{
			"server": map[string]any{
				"host": "profilehost",
			},
		}, "prod")).NotTo(HaveOccurred())
		Expect(c.SetActiveProfiles([]string{"prod"})).NotTo(HaveOccurred())

		Expect(c.InitConfig()).NotTo(HaveOccurred())
		host, err := c.GetConfigValue("server.host")
		Expect(err).NotTo(HaveOccurred())
		Expect(host).To(Equal("profilehost"))

		c.Clear()
		Expect(c.SetActiveProfiles([]string{})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())

		host, err = c.GetConfigValue("server.host")
		Expect(err).NotTo(HaveOccurred())
		Expect(host).To(Equal("basehost"))
	})

	It("does not mutate profile configuration when merging into result", func() {
		c := config.NewConfig()
		Expect(c.LoadConfig(map[string]any{
			"app": map[string]any{
				"env": "base",
			},
		})).NotTo(HaveOccurred())
		Expect(c.LoadConfigForProfile(map[string]any{
			"app": map[string]any{
				"env": "dev",
			},
		}, "dev")).NotTo(HaveOccurred())
		Expect(c.SetActiveProfiles([]string{"dev"})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())

		val, err := c.GetConfigValue("app.env")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("dev"))

		c.Clear()
		Expect(c.SetActiveProfiles([]string{"dev"})).NotTo(HaveOccurred())
		Expect(c.InitConfig()).NotTo(HaveOccurred())

		val, err = c.GetConfigValue("app.env")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("dev"))
	})
})
