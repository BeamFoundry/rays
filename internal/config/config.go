package config

import (
	"encoding"
	"io/fs"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/util"
)

var re = regexp.MustCompile(`(\[[[:word:][:space:].]+\]|\w+)`)

type KeyNotFoundError struct {
	*DefaultError
}

func NewKeyNotFoundError(key, component string) *KeyNotFoundError {
	return &KeyNotFoundError{NewError("KeyNotFoundError", "Key not found", "key", key, "components", component)}
}

type InvalidTypeKeyError struct {
	*DefaultError
}

func NewInvalidTypeKeyError(key, component string) *InvalidTypeKeyError {
	return &InvalidTypeKeyError{NewError("InvalidTypeKeyError", "Key wrong type", "key", key, "components", component)}
}

type configDuplicator struct {
	from    string
	to      string
	profile string
}

type Config struct {
	defaultConfig   HashMap[string, any]
	config          HashMap[string, any]
	profileConfig   HashMap[string, HashMap[string, any]]
	overrides       HashMap[string, string]
	profiles        []string
	defaultConfigFS []fs.FS
	duplicators     []configDuplicator
	configFS        []fs.FS
	lock            sync.Mutex
	initLock        sync.Mutex
	initialized     int32
}

func NewConfigFor(m map[string]any) *Config {
	c := NewConfig()
	//nolint:errcheck // It will never occurre
	c.LoadConfig(m)
	return c
}

// TODO: Define configurable ConfigLoader !!!
func NewConfig() *Config {
	return &Config{
		configFS:        []fs.FS{},
		config:          NewHashMap[string, any](),
		defaultConfigFS: []fs.FS{},
		defaultConfig:   NewHashMap[string, any](),
		duplicators:     []configDuplicator{},
		overrides:       NewHashMap[string, string](),
		profileConfig:   NewHashMap[string, HashMap[string, any]](),
		profiles:        []string{},
	}
}

func (this *Config) Reset() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.profiles = []string{}
	this.config.Clear()
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

func (this *Config) AddConfigFS(fsys fs.FS) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.configFS = append(this.configFS, fsys)
	return nil
}

func (this *Config) AddDefaultConfigFS(fsys fs.FS) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.defaultConfigFS = append(this.defaultConfigFS, fsys)
	return nil
}

func (this *Config) loadDefaultConfig(config map[string]any) {
	this.defaultConfig.Merge(NewHashMapFor(config))
}

func (this *Config) LoadDefaultConfig(config map[string]any) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadDefaultConfig(config)
	return nil
}

func (this *Config) loadDefaultConfigAt(config map[string]any, root string) {
	elems := strings.Split(root, ".")
	for _, e := range slices.Backward(elems) {
		config = map[string]any{e: config}
	}
	this.loadDefaultConfig(config)
}

func (this *Config) LoadDefaultConfigAt(config map[string]any, root string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadDefaultConfigAt(config, root)
	return nil
}

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

func (this *Config) loadConfig(config map[string]any) {
	this.config.Merge(NewHashMapFor(config))
}

func (this *Config) LoadConfig(config map[string]any) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfig(config)
	return nil
}

func (this *Config) loadConfigAt(config map[string]any, root string) {
	elems := strings.Split(root, ".")
	for _, e := range slices.Backward(elems) {
		config = map[string]any{e: config}
	}
	this.loadConfig(config)
}

func (this *Config) LoadConfigAt(config map[string]any, root string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfigAt(config, root)
	return nil
}

func (this *Config) loadConfigForProfile(config map[string]any, profile string) {
	if p, ok := this.profileConfig.Get(profile).Get(); !ok {
		this.profileConfig.Put(profile, NewHashMapFor(config))
	} else {
		p.Merge(NewHashMapFor(config))
	}
}

func (this *Config) LoadConfigForProfile(config map[string]any, profile string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfigForProfile(config, profile)
	return nil
}

func (this *Config) loadConfigForProfileAt(config map[string]any, profile, root string) {
	elems := strings.Split(root, ".")
	for _, e := range slices.Backward(elems) {
		config = map[string]any{e: config}
	}
	this.loadConfigForProfile(config, profile)
}

func (this *Config) LoadConfigForProfileAt(config map[string]any, profile, root string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.loadConfigForProfileAt(config, profile, root)
	return nil
}

func (this *Config) SetActiveProfiles(profiles []string) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}
	this.profiles = profiles
	return nil
}

func (this *Config) ActiveProfiles() []string {
	return this.profiles
}

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
		v, ok := this.config.Get(key).Get()
		if !ok {
			logger.GetLogger().Debug("GetConfigValue: last key failed")
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

	c, err := getMapValue(this.config.Map(), keys)
	if err != nil {
		return NewConfig(), err
	}
	s := NewConfig()
	s.config = NewHashMapFor(c)
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
				loader.ParseConfigurationFilesystem(f, func(m map[string]any, _ string) {
					this.loadDefaultConfig(m)
				})
			}
		}
	}
	for _, l := range configLoaders {
		if loader, ok := reflect.New(l.Elem()).Interface().(ConfigurationParser); ok {
			for _, f := range this.configFS {
				loader.ParseConfigurationFilesystem(f, func(m map[string]any, profile string) {
					if profile == "" {
						this.loadConfig(m)
					} else {
						this.loadConfigForProfile(m, profile)
					}
				})
			}
		}
	}
	config := NewHashMap[string, any]()
	//+ logger.GetLogger().Trace("Config::Pre-Initialize", "default", this.defaultConfig)
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
	//+ logger.GetLogger().Trace("Config::Pre-Initialize", "config", this.config)
	config.Merge(this.config)
	//+ logger.GetLogger().Trace("Config::Post-Initialize", "default", this.defaultConfig)
	this.config = config
	for _, profile := range this.profiles {
		if profileConfig, ok := this.profileConfig.Get(profile).Get(); ok {
			this.config.Merge(profileConfig)
		}
	}
	logger.GetLogger().Debug("Config::Post-Initialize", "config", this.config)
	atomic.StoreInt32(&this.initialized, 1)
	return nil
}

func (this *Config) isInitialized() bool {
	return atomic.CompareAndSwapInt32(&this.initialized, 1, 1)
}

func (this *Config) validate(obj any) Error {
	return Validate(obj)
}

func (this *Config) unmarshal(obj any) Error {
	convert := func(elem any, rt reflect.Type) (reflect.Value, Error) {
		var value reflect.Value
		t := rt

		if t.Kind() == reflect.Pointer {
			if elem == nil {
				return value, nil
			}
			t = t.Elem()
		}

		val, err := Convert(elem, t)
		if err != nil && err.Name() != "ConvertionError" {
			return value, err
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
					s.config.PutAll(NewHashMapFor(c))
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
				for k := range s.config.Iterator() {
					ptr := reflect.New(field.Type.Elem().Elem())
					if e, err := s.GetSubConfig("[" + k + "]"); err == nil {
						if err := e.unmarshal(ptr.Interface()); err != nil {
							return err
						}
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
		}
		var value reflect.Value
		override := String(field.Name).ToLowerCase().String()

		if v, ok := this.config.Get(field.Name).Get(); ok {
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
