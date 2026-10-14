// # Initialization
//
// This package let you configure and initialize the IoC container. It is meant used from a main() function.
//
// The configuration is done through a call to the [ApplicationContext] function that let you retrieve a
// [github.com/BeamFoundry/rays/pkg/core.ContainerConfiguration].
//
//	func main() {
//	    ctx := rays.ApplicationContext()
//
//	    if err := ctx.Initialize(); err != nil {
//	        err.LogErrorWithStackTrace()
//	        return
//	    }
//
//	    ctx.Run()
//	}
//
// # Packages
//
// This project divides its code between the following packages :
//
//   - [github.com/BeamFoundry/rays/pkg/core]: Contains interfaces definitions and basic objects directly linked
//     to the framework internals.
//   - [github.com/BeamFoundry/rays/pkg/lang]: Contains language advance object types here to leverage the Go
//     language like [github.com/BeamFoundry/rays/pkg/lang.Error].
//   - [github.com/BeamFoundry/rays/pkg/util]: Contains more language advance object types, some not yet ready to
//     be promoted to the [github.com/BeamFoundry/rays/pkg/lang] package.
//   - [github.com/BeamFoundry/rays/pkg/i18n]: Contains location and internationalization object.
//   - [github.com/BeamFoundry/rays/pkg/web]: Contains web related interfaces and objects.
//
// # Sample main file
//
// This is a sample default main() function that let you fully initialize an application with Rays.
//
// It defines three flags to:
//
//   - Set active profiles (-p|--profile profile1,profile2)
//   - Add local directory to load configuration files from (-c|--config ./config)
//   - Override a configuration parameter (--override rays.logging.level=DEBUG)
//
// All flags can be used multiple times and/or values can be comma-separated :
//
//	./app --profile testing,sqlite --profile admin
//
// The override flag let you define key=value pairs to replace configuration values.
// It use the same dot separated format for the key hierarchie as ConfigurationProperties
//
// For example, for a yaml configuration :
//
//	app:
//	  module:
//	    value: foo
//
// To replace the inner "value" key, you should define an override like :
//
//	--override app.module.config.value=bar
//
// The "overrides" system is designed to work with string values even if the target value is a bool or numeric value.
//
// Additionally it will add overrides from environment variables whose name starts by "RAYS" or "APP",
// lowering the case and replacing '_' with '.'.
//
//	package main
//
//	import (
//	    // Stdlib packages
//	    "os"
//
//	    // 3rd-Party packages for main
//	    flag "github.com/spf13/pflag"
//
//	    // Rays
//	    "github.com/BeamFoundry/rays"
//
//	    // Importing rays/lang
//	    . "github.com/BeamFoundry/rays/pkg/lang"
//
//	    // Initializing rays external modules
//	    _ "github.com/BeamFoundry/rays-fiber"
//	    _ "github.com/BeamFoundry/rays-orm"
//	    _ "github.com/BeamFoundry/rays-web-security"
//
//	    // Importing embedded resources
//	    "myrepo/myapp/resources"
//
//	    // Initializing local packages
//	    _ "myrepo/myapp/pkg/configurations"
//	    _ "myrepo/myapp/pkg/controllers"
//	    _ "myrepo/myapp/pkg/domains"
//	    _ "myrepo/myapp/pkg/repositories"
//	    _ "myrepo/myapp/pkg/services"
//	)
//
//	var (
//	    config   []string = []string{}
//	    profiles []string = []string{}
//	    overrides map[string]string = map[string]string{}
//	)
//
//	func main() {
//	    // Adding flags
//	    flag.StringSliceVarP(&config, "config", "c", []string{}, "Additional configuration paths")
//	    flag.StringSliceVarP(&profiles, "profile", "p", []string{}, "Profile to use, default : none")
//	    flag.StringToStringVar(&overrides, "override", map[string]string{}, "Overrides configuration key inline")
//	    flag.Parse()
//
//	    // Getting rays ApplicationContext
//	    ctx := rays.ApplicationContext()
//
//	    // Loading embedded configuration
//	    ctx.AddConfigFS(resources.Config())
//
//	    // Set profiles
//	    if len(profiles) > 0 {
//	        ctx.SetActiveProfiles(profiles)
//	    }
//
//	    // Add additional local config directory
//	    for _, dir := range config {
//	        if stat, err := os.Stat(dir); os.IsExist(err) && stat.IsDir() {
//	            ctx.AddConfigFS(os.DirFS(dir))
//	        }
//	    }
//
//	    // Load any overrides defined
//	    for key, value := range overrides {
//	        ctx.AddOverride(key, value)
//	    }
//
//	    // Load overrides with environment variables
//	    //
//	    // Only adding environment variables keys starting with 'RAYS_' or 'APP_',
//	    // lowering the case and replacing '_' with '.' .
//	    for _, e := range os.Environ() {
//	        env := Strings(e).SplitN("=", 2)
//	        key := String(env[0]).ToUpperCase()
//	        if key.StartsWith("RAYS_") || key.StartsWith("APP_") {
//	            key = key.Replace("_", ".").ToLowerCase()
//	            ctx.AddOverride(key.String(), env[1])
//	        }
//	    }
//
//	    // Initialize the container
//	    if err := ctx.Initialize(); err != nil {
//	        panic(err)
//	    }
//
//	    // Find and run "Runnables"
//	    ctx.Run()
//	}
package rays

import (
	"github.com/BeamFoundry/rays/internal/container"
	"github.com/BeamFoundry/rays/pkg/core"
)

// ApplicationContext() returns a [core.ContainerConfiguration], letting you control Rays container initialization.
func ApplicationContext() core.ContainerConfiguration {
	return container.GetContainer()
}
