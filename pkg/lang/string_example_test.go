//nolint:errcheck // Not printing error handling
package lang_test

import (
	"fmt"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

func ExampleString_Format() {
	fmt.Println(String("%s : %.2f").Format("Pi", 3.14159))
	// Output: Pi : 3.14
}
