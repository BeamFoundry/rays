package container

import (
	"reflect"
	"regexp"
	"strings"
)

//nolint:gocritic // false positive
var reTag = regexp.MustCompile(`([A-Z][A-Za-z]*)(?:{([^}]*)}|)`)

type AnnotationField struct {
	field      *reflect.StructField
	attributes map[string]string
	options    map[string]string
	inject     bool
}

func NewAnnotationField(field *reflect.StructField) *AnnotationField {
	annotation, ok := field.Tag.Lookup("@")
	if !ok {
		return nil
	}
	annotated := &AnnotationField{
		field:      field,
		attributes: map[string]string{},
		options:    map[string]string{},
	}
	components := reTag.FindAllStringSubmatch(annotation, -1)
	for _, component := range components {
		switch {
		case strings.EqualFold(component[1], "Autowired"),
			strings.EqualFold(component[1], "Inject"):
			annotated.inject = true
			if options := component[2]; options != "" {
				for option := range strings.SplitSeq(options, ",") {
					if kv := strings.Split(option, "="); len(kv) == 2 {
						annotated.options[kv[0]] = kv[1]
					}
				}
			}
		default:
			annotated.attributes[component[1]] = component[2]
		}
	}
	return annotated
}

func (this *AnnotationField) Attributes() map[string]string {
	return this.attributes
}

func (this *AnnotationField) Options() map[string]string {
	return this.options
}

func (this *AnnotationField) Inject() bool {
	return this.inject
}
