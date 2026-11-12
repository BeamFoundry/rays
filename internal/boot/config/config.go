// Package config provides hierarchical application configuration loading,
// profile merging, overrides, type conversion, unmarshaling, and validation for Rays.
package config

import (
	"encoding"
	"fmt"
	"io/fs"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/BeamFoundry/rays/internal/boot/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/util"
)

// re matches configuration path segments, supporting bracketed identifiers (e.g. "[foo.bar]") and words.
var re = regexp.MustCompile(`(\[[[:word:][:space:].]+\]|\w+)`)

// KeyNotFoundError represents an error when a requested configuration key path does not exist.
type KeyNotFoundError struct {
	*DefaultError
}

// NewKeyNotFoundError constructs a KeyNotFoundError for the given key and failing path component.
func NewKeyNotFoundError(key, component string) *KeyNotFoundError {
	return &KeyNotFoundError{NewError("KeyNotFoundError", "Key not found", "key", key, "components", component)}
}

// InvalidTypeKeyError represents an error when traversing a key path encounters an incompatible data type.
type InvalidTypeKeyError struct {
	*DefaultError
}

// NewInvalidTypeKeyError constructs an InvalidTypeKeyError for the given key and incompatible path component.
func NewInvalidTypeKeyError(key, component string) *InvalidTypeKeyError {
	return &InvalidTypeKeyError{NewError("InvalidTypeKeyError", "Key wrong type", "key", key, "components", component)}
}

// configDuplicator defines a configuration duplication rule from a source path to a destination path,
// optionally scoped to an active profile.
type configDuplicator struct {
	from    string
	to      string
	profile string
}

// Config manages hierarchical configuration data, profiles, defaults, overrides,
// and unmarshaling into strongly-typed structures.
type Config struct {
	logger          *logger.Logger
	config          HashMap[string, any]
	profileConfig   HashMap[string, HashMap[string, any]]
	overrides       HashMap[string, string]
	result          HashMap[string, any]
	defaultConfig   HashMap[string, any]
	profiles        []string
	defaultConfigFS []fs.FS
	duplicators     []configDuplicator
	configFS        []fs.FS
	lock            sync.Mutex
	initLock        sync.Mutex
	initialized     int32
}

// NewConfigFor constructs and returns a new Config preloaded with the given configuration map.
func NewConfigFor(m map[string]any) *Config {
	c := NewConfig()
	//nolint:errcheck // It will never occurre
	c.LoadConfig(m)
	return c
}

// newConfigWithLogger creates an uninitialized Config instance sharing or using the specified logger.
func newConfigWithLogger(l *logger.Logger) *Config {
	return &Config{
		configFS:        []fs.FS{},
		config:          NewHashMap[string, any](),
		defaultConfigFS: []fs.FS{},
		defaultConfig:   NewHashMap[string, any](),
		duplicators:     []configDuplicator{},
		overrides:       NewHashMap[string, string](),
		profileConfig:   NewHashMap[string, HashMap[string, any]](),
		profiles:        []string{},
		result:          NewHashMap[string, any](),
		logger:          l,
	}
}

// NewConfig initializes and returns an empty Config container with a default logger.
func NewConfig() *Config {
	return newConfigWithLogger(logger.NewLogger())
}

// SetLogger replaces the logger instance used by the configuration container if non-nil.
func (this *Config) SetLogger(l *logger.Logger) {
	if l != nil {
		this.logger = l
	}
}

// Clear clears the cached merged configuration results and resets the initialization state,
// allowing the configuration container to be re-initialized.
func (this *Config) Clear() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.result.Clear()
	atomic.StoreInt32(&this.initialized, 0)
}

// getActiveConfig returns the active configuration map: the merged result map if initialized,
// or the raw config map otherwise.
func (this *Config) getActiveConfig() HashMap[string, any] {
	if this.result.Size() > 0 {
		return this.result
	}
	return this.config
}

// Reset clears all configurations, profiles, overrides, registered filesystems, and cached state.
func (this *Config) Reset() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.profiles = []string{}
	this.config.Clear()
	this.result.Clear()
	this.overrides.Clear()
	this.configFS = []fs.FS{}
	this.defaultConfig.Remove("test")
	this.defaultConfigFS = slices.DeleteFunc(this.defaultConfigFS, func(f fs.FS) bool {
		info, err := fs.Stat(f, "testdata")

		// If there is no error and the file is a directory, return true to DELETE it.
		return err == nil && info.IsDir()
	})
	this.initialized = 0
}

// AddOverride registers a key-value override that takes precedence over loaded configuration values.
// Returns an error if the container has already been initialized.
func (this *Config) AddOverride(key, value string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	keys := []string{}
	for _, match := range re.FindAllStringSubmatch(key, -1) {
		if _, err := strconv.Atoi(match[1]); err == nil {
			keys = append(keys, "["+match[1]+"]")
			continue
		}
		keys = append(keys, match[1])
	}
	k := strings.Join(keys, ".")
	this.overrides.Put(String(k).ToLowerCase().String(), value)
	return nil
}

// AddConfigFS registers a filesystem to load configuration files from during initialization.
// Returns an error if the container has already been initialized.
func (this *Config) AddConfigFS(fsys fs.FS) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.configFS = append(this.configFS, fsys)
	return nil
}

// AddDefaultConfigFS registers a filesystem providing default fallback configuration files.
// Returns an error if the container has already been initialized.
func (this *Config) AddDefaultConfigFS(fsys fs.FS) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.defaultConfigFS = append(this.defaultConfigFS, fsys)
	return nil
}

// loadDefaultConfig merges raw default configuration values into the default configuration map.
func (this *Config) loadDefaultConfig(config map[string]any) {
	this.defaultConfig.Merge(NewHashMapFor(config))
}

// LoadDefaultConfig merges default fallback configuration values into the container.
// Returns an error if the container has already been initialized.
func (this *Config) LoadDefaultConfig(config map[string]any) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadDefaultConfig(config)
	return nil
}

// loadDefaultConfigAt merges default configuration values nested under a dot-delimited root key path.
func (this *Config) loadDefaultConfigAt(config map[string]any, root string) {
	elems := strings.Split(root, ".")
	for _, e := range slices.Backward(elems) {
		config = map[string]any{e: config}
	}
	this.loadDefaultConfig(config)
}

// LoadDefaultConfigAt merges default configuration values nested under a dot-delimited root key path.
// Returns an error if the container has already been initialized.
func (this *Config) LoadDefaultConfigAt(config map[string]any, root string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadDefaultConfigAt(config, root)
	return nil
}

// CopyDefaultConfig registers a duplication rule copying default configuration from one path to another.
// Returns an error if the container has already been initialized.
func (this *Config) CopyDefaultConfig(from, to string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.duplicators = append(this.duplicators, configDuplicator{
		from: from,
		to:   to,
	})
	return nil
}

// CopyDefaultConfigForProfile registers a duplication rule copying default configuration for a specific profile.
// Returns an error if the container has already been initialized.
func (this *Config) CopyDefaultConfigForProfile(from, to, profile string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.duplicators = append(this.duplicators, configDuplicator{
		from:    from,
		to:      to,
		profile: profile,
	})
	return nil
}

// loadConfig merges configuration entries into the raw configuration map and active result map.
func (this *Config) loadConfig(config map[string]any) {
	this.config.Merge(NewHashMapFor(config))
	if this.result.Size() > 0 {
		this.result.Merge(NewHashMapFor(deepCloneMap(config)))
	}
}

// LoadConfig merges application configuration values into the container.
// Returns an error if the container has already been initialized.
func (this *Config) LoadConfig(config map[string]any) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfig(config)
	return nil
}

// loadConfigAt merges configuration values nested under a dot-delimited root key path.
func (this *Config) loadConfigAt(config map[string]any, root string) {
	elems := strings.Split(root, ".")
	for _, e := range slices.Backward(elems) {
		config = map[string]any{e: config}
	}
	this.loadConfig(config)
}

// LoadConfigAt merges configuration values nested under a dot-delimited root key path.
// Returns an error if the container has already been initialized.
func (this *Config) LoadConfigAt(config map[string]any, root string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfigAt(config, root)
	return nil
}

// loadConfigForProfile merges configuration values under the specified profile name.
func (this *Config) loadConfigForProfile(config map[string]any, profile string) {
	if p, ok := this.profileConfig.Get(profile).Get(); !ok {
		this.profileConfig.Put(profile, NewHashMapFor(config))
	} else {
		p.Merge(NewHashMapFor(config))
	}
}

// LoadConfigForProfile merges configuration values for a specific profile into the container.
// Returns an error if the container has already been initialized.
func (this *Config) LoadConfigForProfile(config map[string]any, profile string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfigForProfile(config, profile)
	return nil
}

// loadConfigForProfileAt merges configuration values for a profile nested under a dot-delimited root key path.
func (this *Config) loadConfigForProfileAt(config map[string]any, profile, root string) {
	elems := strings.Split(root, ".")
	for _, e := range slices.Backward(elems) {
		config = map[string]any{e: config}
	}
	this.loadConfigForProfile(config, profile)
}

// LoadConfigForProfileAt merges configuration values for a profile nested under a dot-delimited root key path.
// Returns an error if the container has already been initialized.
func (this *Config) LoadConfigForProfileAt(config map[string]any, profile, root string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfigForProfileAt(config, profile, root)
	return nil
}

// SetActiveProfiles configures the list of active profile names to activate during initialization.
// Returns an error if the container has already been initialized.
func (this *Config) SetActiveProfiles(profiles []string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.profiles = profiles
	return nil
}

// ActiveProfiles returns the slice of active configuration profile names.
func (this *Config) ActiveProfiles() []string {
	return this.profiles
}

// GetConfigValue retrieves a string configuration value by dot-notated key path,
// applying registered overrides and converting the value to string.
func (this *Config) GetConfigValue(key string) (string, Error) {
	keys := []string{}
	for _, match := range re.FindAllStringSubmatch(key, -1) {
		keys = append(keys, match[1])
	}

	if len(keys) == 1 {
		k := keys[0]
		k, _ = strings.CutPrefix(k, "[")
		k, _ = strings.CutSuffix(k, "]")
		override := String(k).ToLowerCase().String()
		v, ok := this.getActiveConfig().Get(key).Get()
		if !ok {
			this.logger.Debug("GetConfigValue: last key failed")
			return "", NewKeyNotFoundError(key, keys[0])
		}
		if o, ok := this.overrides.Get(override).Get(); ok {
			v = o
		}
		val, err := Convert(v, reflect.TypeFor[string]())
		if err != nil {
			return "", err
		}
		return val.String(), nil
	}
	c, err := this.GetSubConfig(strings.Join(keys[:len(keys)-1], "."))
	if err != nil {
		err.LogError()
		return "", err
	}

	return c.GetConfigValue(keys[len(keys)-1])
}

// Unmarshal maps active configuration values into the fields of the target struct pointer
// and validates the resulting structure using defined validation tags.
func (this *Config) Unmarshal(obj any) Error {
	t := reflect.TypeOf(obj)

	if t.Kind() != reflect.Pointer {
		return NewError("configError", "object must be a pointer")
	}

	if err := this.unmarshal(obj); err != nil {
		err.LogErrorWithStackTrace()
		return err
	}

	return this.validate(obj)
}

// deepCloneMap returns an independent deep copy of the provided string-to-any map,
// cloning any nested maps or slices recursively.
func deepCloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = deepCloneValue(v)
	}
	return cp
}

// deepCloneSlice returns an independent deep copy of the provided slice,
// cloning any nested maps or slices recursively.
func deepCloneSlice(s []any) []any {
	if s == nil {
		return nil
	}
	cp := make([]any, len(s))
	for i, v := range s {
		cp[i] = deepCloneValue(v)
	}
	return cp
}

// deepCloneValue returns a deep copy of v if v is a map[string]any or []any,
// or returns v unaltered if it is an immutable scalar type.
func deepCloneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCloneMap(val)
	case []any:
		return deepCloneSlice(val)
	default:
		return v
	}
}

// GetSubConfig extracts and returns an isolated sub-configuration rooted at the specified
// dot-separated key path, deep-cloning backing maps and propagating matching prefix overrides.
func (this *Config) GetSubConfig(key string) (*Config, Error) {
	keys := []string{}
	for _, match := range re.FindAllStringSubmatch(key, -1) {
		keys = append(keys, match[1])
	}
	var getMapValue func(m map[string]any, keys []string) (map[string]any, Error)

	getMapValue = func(m map[string]any, keys []string) (map[string]any, Error) {
		k := keys[0]
		k, _ = strings.CutPrefix(k, "[")
		k, _ = strings.CutSuffix(k, "]")
		if v, ok := m[k]; !ok {
			return nil, NewKeyNotFoundError(keys[0], key)
		} else {
			if val, ok := v.(map[string]any); ok {
				if len(keys) > 1 {
					return getMapValue(val, keys[1:])
				}
				return val, nil
			}
			if val, ok := v.([]any); ok {
				if len(keys) == 1 {
					// Unmarshal will always show the last key
					return map[string]any{k: val}, nil
				}
				k = keys[1]
				k, _ = strings.CutPrefix(k, "[")
				k, _ = strings.CutSuffix(k, "]")
				if idx, err := strconv.Atoi(k); err == nil {
					// GetConfigValue need this.
					return getMapValue(map[string]any{k: val[idx]}, keys[1:])
				}
				return nil, NewInvalidTypeKeyError(keys[0], key)
			}
		}
		return nil, NewInvalidTypeKeyError(keys[0], key)
	}

	c, err := getMapValue(this.getActiveConfig().Map(), keys)
	if err != nil {
		return newConfigWithLogger(this.logger), err
	}
	s := newConfigWithLogger(this.logger)
	s.config = NewHashMapFor(deepCloneMap(c))
	s.result = NewHashMapFor(deepCloneMap(c))
	if len(keys) == 1 {
		key, _ = strings.CutPrefix(key, "[")
		key, _ = strings.CutSuffix(key, "]")
	}
	prefix := String(key).ToLowerCase().String() + "."
	for k, v := range this.overrides.Iterator() {
		if p := String(k); p.StartsWith(prefix) {
			s.overrides.Put(Strings(p).TrimPrefix(prefix), v)
		}
	}
	return s, nil
}

// InitConfig executes configuration initialization:
// 1. Invokes all discovered ConfigurationParser instances on default and application filesystems.
// 2. Merges default configurations, copies duplicated subtrees, and merges base configs.
// 3. Merges active profile configurations into the final result map.
//
// Returns an error if the container has already been initialized.
func (this *Config) InitConfig() Error {
	this.initLock.Lock()
	defer this.initLock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	// 1. Loading default [core.ConfigurationParser]
	// configLoaders := []reflect.Type{reflect.TypeFor[*YamlConfigFSLoader]()}
	configLoaders := FindTypesImplementingInterface(reflect.TypeFor[ConfigurationParser]())
	for _, l := range configLoaders {
		if loader, ok := reflect.New(l.Elem()).Interface().(ConfigurationParser); ok {
			for _, f := range this.defaultConfigFS {
				if err := loader.ParseConfigurationFilesystem(f, func(m map[string]any, _ string) {
					this.loadDefaultConfig(m)
				}); err != nil {
					return err
				}
			}
		}
	}
	for _, l := range configLoaders {
		if loader, ok := reflect.New(l.Elem()).Interface().(ConfigurationParser); ok {
			for _, f := range this.configFS {
				if err := loader.ParseConfigurationFilesystem(f, func(m map[string]any, profile string) {
					if profile == "" {
						this.loadConfig(m)
					} else {
						this.loadConfigForProfile(m, profile)
					}
				}); err != nil {
					return err
				}
			}
		}
	}
	config := NewHashMap[string, any]()
	config.Merge(this.defaultConfig)
	for _, duplicate := range this.duplicators {
		err := Try(func() {
			source := RaiseVal(NewConfigFor(config.Map()).GetSubConfig(duplicate.from))
			if duplicate.profile == "" {
				this.loadConfigAt(source.config.Map(), duplicate.to)
			} else {
				this.loadConfigForProfileAt(source.config.Map(), duplicate.profile, duplicate.to)
			}
			RaiseNilErr(nil)
		})
		if err != nil {
			err.LogError()
		}
	}
	baseConfig := NewHashMapFor(deepCloneMap(this.config.Map()))
	config.Merge(baseConfig)
	this.result = config
	for _, profile := range this.profiles {
		if profileConfig, ok := this.profileConfig.Get(profile).Get(); ok {
			this.result.Merge(NewHashMapFor(deepCloneMap(profileConfig.Map())))
		}
	}
	this.logger.Debug("Config::Post-Initialize", "config", this.result)
	atomic.StoreInt32(&this.initialized, 1)
	return nil
}

// isInitialized reports whether InitConfig has already been successfully executed on this container.
func (this *Config) isInitialized() bool {
	return atomic.CompareAndSwapInt32(&this.initialized, 1, 1)
}

// validate runs struct tag validations on the target struct using the validator framework.
func (this *Config) validate(obj any) Error {
	return Validate(obj)
}

// unmarshal recursively maps nested configuration values into the target struct pointer's
// fields, slices, and maps, applying custom type conversions and dot-notated key overrides.
func (this *Config) unmarshal(obj any) Error {
	convert := func(elem any, rt reflect.Type) (reflect.Value, Error) {
		var value reflect.Value
		t := rt

		if t.Kind() == reflect.Pointer {
			if elem == nil {
				return reflect.Zero(rt), nil
			}
			t = t.Elem()
		}

		val, err := Convert(elem, t)
		if err != nil {
			if err.Name() != "ConvertionError" || err.Message() != "Not a valid Type Conversion" {
				return value, err
			}

			v := reflect.ValueOf(elem)
			if !v.IsValid() || !v.CanConvert(t) {
				return value, NewError("ConvertionError", fmt.Sprintf("cannot convert value of type %T to %v", elem, t))
			}

			if err := checkNumericRange(v, t); err != nil {
				return value, err
			}

			val = v.Convert(t)
		}
		if rt.Kind() == reflect.Pointer {
			v := reflect.New(t)
			v.Elem().Set(val)
			return v, nil
		}
		return val, nil
	}
	t := reflect.TypeOf(obj)

	for i := range t.Elem().NumField() {
		field := t.Elem().Field(i)
		f := UnprotectValue(reflect.ValueOf(obj).Elem().Field(i))
		if field.Type.Kind() == reflect.Struct &&
			!f.Addr().Type().Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) {
			if s, err := this.GetSubConfig("[" + field.Name + "]"); err == nil {
				if err := s.unmarshal(f.Addr().Interface()); err != nil {
					return err
				}
			}
			continue
		}
		if field.Type.Kind() == reflect.Pointer &&
			field.Type.Elem().Kind() == reflect.Struct &&
			!field.Type.Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) &&
			!field.Type.Elem().Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) {
			if _, ok := this.getActiveConfig().Get(field.Name).Get(); ok {
				s, err := this.GetSubConfig("[" + field.Name + "]")
				if err != nil {
					return err
				}
				ptr := reflect.New(field.Type.Elem())
				if err := s.unmarshal(ptr.Interface()); err != nil {
					return err
				}
				f.Set(ptr)
			}
			continue
		}
		if field.Type.Kind() == reflect.Slice {
			key := field.Name
			subkey, err := this.GetSubConfig("[" + key + "]")
			if err != nil {
				continue
			}
			// GetSubConfig will create the key, no need to check for it.
			cfg, _ := subkey.config.Get(key).Get()

			vals, ok := cfg.([]any)
			if !ok {
				continue
			}
			slice := reflect.MakeSlice(field.Type, 0, len(vals))
			for i, v := range vals {
				if field.Type.Elem().Kind() == reflect.Struct {
					c, ok := v.(map[string]any)
					if !ok {
						continue
					}
					s := NewConfig()
					_, hasKey := field.Type.Elem().FieldByName("key")
					_, hasValue := field.Type.Elem().FieldByName("value")
					mapEntry := Strings(field.Type.Elem().String()).HasPrefix("util.MapEntry[")
					if len(c) == 1 && hasKey && hasValue && mapEntry {
						for entryKey, entryValue := range c {
							s.config.Put("key", entryKey)
							s.config.Put("value", entryValue)
						}
					} else {
						s.config.PutAll(NewHashMapFor(c))
					}
					prefix := String(key).ToLowerCase().String() + ".[" + strconv.Itoa(i) + "]."
					for k, v := range this.overrides.Iterator() {
						if p := String(k); p.StartsWith(prefix) {
							s.overrides.Put(Strings(p).TrimPrefix(prefix), v)
						}
					}
					e := reflect.New(field.Type.Elem())
					if err := s.unmarshal(e.Interface()); err != nil {
						return err
					}
					slice = reflect.Append(slice, e.Elem())
				} else if field.Type.Elem().Kind() == reflect.Map {
					c, ok := v.(map[string]any)
					if !ok {
						continue
					}
					s := NewConfig()
					s.config.PutAll(NewHashMapFor(c))
					prefix := String(key).ToLowerCase().String() + ".[" + strconv.Itoa(i) + "]."
					for k, v := range this.overrides.Iterator() {
						if p := String(k); p.StartsWith(prefix) {
							o := Strings(p).TrimPrefix(prefix)
							s.overrides.Put(o, v)
						}
					}
					m := reflect.MakeMapWithSize(field.Type.Elem(), s.config.Size())

					for k, v := range s.config.Iterator() {
						kval := reflect.ValueOf(k)
						if kv, err := convert(k, field.Type.Elem().Key()); err == nil {
							kval = kv
						}
						override := String(k).ToLowerCase().String()
						if o, ok := s.overrides.Get(override).Get(); ok {
							v = o
						}
						val, err := convert(v, field.Type.Elem().Elem())
						if err != nil {
							return err
						}
						m.SetMapIndex(kval, val)
					}
					slice = reflect.Append(slice, m)
				} else if val, err := convert(v, field.Type.Elem()); err != nil {
					return err
				} else {
					slice = reflect.Append(slice, val)
				}
			}
			f.Set(slice)
			continue
		}
		if field.Type.Kind() == reflect.Map {
			s, err := this.GetSubConfig(field.Name)
			if err != nil {
				continue
			}
			m := reflect.MakeMapWithSize(field.Type, s.config.Size())
			p := reflect.PointerTo(field.Type.Elem())
			switch {
			case field.Type.Elem().Kind() == reflect.Struct &&
				!p.Implements(reflect.TypeFor[encoding.TextUnmarshaler]()):
				for k := range s.config.Iterator() {
					ptr := reflect.New(field.Type.Elem())
					if e, err := s.GetSubConfig("[" + k + "]"); err == nil {
						if err := e.unmarshal(ptr.Interface()); err != nil {
							return err
						}
						m.SetMapIndex(reflect.ValueOf(k), ptr.Elem())
					}
				}
			case field.Type.Elem().Kind() == reflect.Pointer &&
				field.Type.Elem().Elem().Kind() == reflect.Struct &&
				!field.Type.Elem().Elem().Implements(reflect.TypeFor[encoding.TextUnmarshaler]()):
				for k, v := range s.config.Iterator() {
					if v == nil {
						m.SetMapIndex(reflect.ValueOf(k), reflect.Zero(field.Type.Elem()))
						continue
					}
					e, err := s.GetSubConfig("[" + k + "]")
					if err != nil {
						return err
					}
					ptr := reflect.New(field.Type.Elem().Elem())
					if err := e.unmarshal(ptr.Interface()); err != nil {
						return err
					}
					m.SetMapIndex(reflect.ValueOf(k), ptr)
				}
			case field.Type.Elem().Kind() == reflect.Map:
				for k := range s.config.Iterator() {
					//nolint:errcheck // Iterating valid key.
					e, _ := s.GetSubConfig("[" + k + "]")
					mapsub := NewConfigFor(map[string]any{
						"Map": e.config.Map(),
					})
					fields := []reflect.StructField{
						{
							Name: "Map",
							Type: field.Type.Elem(),
						},
					}
					obj := reflect.StructOf(fields)
					ptr := reflect.New(obj)
					if err := mapsub.unmarshal(ptr.Interface()); err == nil {
						val := ptr.Elem().Field(0)
						m.SetMapIndex(reflect.ValueOf(k), val)
					}
				}
			default:
				for k, v := range s.config.Iterator() {
					kval := reflect.ValueOf(k)
					if kv, err := convert(k, field.Type.Key()); err == nil {
						kval = kv
					}
					val, err := convert(v, field.Type.Elem())
					if err != nil {
						return err
					}
					m.SetMapIndex(kval, val)
				}
			}
			f.Set(m)
			continue
		}
		var value reflect.Value
		override := String(field.Name).ToLowerCase().String()

		if v, ok := this.getActiveConfig().Get(field.Name).Get(); ok {
			if o, ok := this.overrides.Get(override).Get(); ok {
				v = o
			}
			val, err := convert(v, field.Type)
			if err != nil {
				return err
			}
			value = val
		}

		if value.IsValid() {
			f.Set(value)
		}
	}
	return nil
}

// checkNumericRange validates that a numeric value v can be represented within the bounds of target type t.
func checkNumericRange(v reflect.Value, t reflect.Type) Error {
	vKind := v.Kind()
	tKind := t.Kind()

	switch {
	case isSignedInt(tKind):
		return checkSignedTargetRange(v, vKind, t, tKind)
	case isUnsignedInt(tKind):
		return checkUnsignedTargetRange(v, vKind, t, tKind)
	case isFloat(tKind):
		return checkFloatTargetRange(v, vKind, tKind)
	default:
		return nil
	}
}

func isSignedInt(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Int64
}

func isUnsignedInt(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}

func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

func checkSignedTargetRange(v reflect.Value, vKind reflect.Kind, t reflect.Type, tKind reflect.Kind) Error {
	var minInt64, maxInt64 int64
	switch tKind {
	case reflect.Int8:
		minInt64, maxInt64 = math.MinInt8, math.MaxInt8
	case reflect.Int16:
		minInt64, maxInt64 = math.MinInt16, math.MaxInt16
	case reflect.Int32:
		minInt64, maxInt64 = math.MinInt32, math.MaxInt32
	case reflect.Int64:
		minInt64, maxInt64 = math.MinInt64, math.MaxInt64
	default:
		minInt64, maxInt64 = math.MinInt, math.MaxInt
	}

	if isSignedInt(vKind) {
		val := v.Int()
		if val < minInt64 || val > maxInt64 {
			return NewError("ConvertionError", fmt.Sprintf("integer value %d out of range for target type %v", val, t))
		}
	}
	if isUnsignedInt(vKind) {
		val := v.Uint()
		if maxInt64 < 0 || val > uint64(maxInt64) {
			return NewError("ConvertionError", fmt.Sprintf("unsigned integer value %d out of range for signed target type %v", val, t))
		}
	}
	if isFloat(vKind) {
		val := v.Float()
		if val < float64(minInt64) || val > float64(maxInt64) {
			return NewError("ConvertionError", fmt.Sprintf("float value %g out of range for target type %v", val, t))
		}
	}
	return nil
}

func checkUnsignedTargetRange(v reflect.Value, vKind reflect.Kind, t reflect.Type, tKind reflect.Kind) Error {
	var maxUint64 uint64
	switch tKind {
	case reflect.Uint8:
		maxUint64 = math.MaxUint8
	case reflect.Uint16:
		maxUint64 = math.MaxUint16
	case reflect.Uint32:
		maxUint64 = math.MaxUint32
	case reflect.Uint64:
		maxUint64 = math.MaxUint64
	default:
		maxUint64 = math.MaxUint
	}

	if isSignedInt(vKind) {
		val := v.Int()
		if val < 0 {
			return NewError("ConvertionError", fmt.Sprintf("negative integer value %d cannot be converted to unsigned target type %v", val, t))
		}
		if uint64(val) > maxUint64 {
			return NewError("ConvertionError", fmt.Sprintf("integer value %d out of range for unsigned target type %v", val, t))
		}
	}
	if isUnsignedInt(vKind) {
		val := v.Uint()
		if val > maxUint64 {
			return NewError("ConvertionError", fmt.Sprintf("unsigned integer value %d out of range for unsigned target type %v", val, t))
		}
	}
	if isFloat(vKind) {
		val := v.Float()
		if val < 0 || val > float64(maxUint64) {
			return NewError("ConvertionError", fmt.Sprintf("float value %g out of range for unsigned target type %v", val, t))
		}
	}
	return nil
}

func checkFloatTargetRange(v reflect.Value, vKind, tKind reflect.Kind) Error {
	if tKind == reflect.Float32 && isFloat(vKind) {
		val := v.Float()
		if math.Abs(val) > math.MaxFloat32 {
			return NewError("ConvertionError", fmt.Sprintf("float value %g overflows float32", val))
		}
	}
	return nil
}
