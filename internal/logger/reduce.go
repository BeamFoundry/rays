package logger

import (
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
)

var moduleCache = map[string]string{}

var functionCache = map[string]string{}

var functionLock sync.Mutex

// Given a slice of string, fisrt will return a string with all but last element reduced
// to its reducePkgElements letter and the last element returned in its entirety, all separeted by a slash.
//
// Example : `[]string{"github.com","BeamFoundry","rays"}` will return `"g/B/rays"`.
func reducePkgElements(elems []string) string {
	if len(elems) == 1 {
		return elems[0]
	}
	var s string
	for _, e := range elems[0 : len(elems)-1] {
		s = s + string(e[0]) + "/"
	}
	return s + elems[len(elems)-1]
}

// ReduceFunctionName reduces a fully qualified function name for concise log source reporting.
func ReduceFunctionName(function string) string {
	if !strings.Contains(function, ".") {
		return function
	}

	functionLock.Lock()
	defer functionLock.Unlock()

	if ret, ok := functionCache[function]; ok {
		return ret
	}

	dir := filepath.Dir(function)
	base := filepath.Base(function)
	comp := strings.SplitN(base, ".", 3)
	if len(comp) < 2 || comp[0] == "" || comp[1] == "" {
		return function
	}

	obj := comp[1]
	if strings.HasPrefix(obj, "(*") {
		if len(obj) < 4 || !strings.HasSuffix(obj, ")") {
			return function
		}
		obj = obj[2 : len(obj)-1]
	}
	obj = strings.Replace(obj, "[...]", "", 1)

	pkg := dir + "/" + comp[0]
	for k, v := range moduleCache {
		if k == pkg {
			pkg = v
		} else if strings.HasPrefix(pkg, k) {
			if s, ok := strings.CutPrefix(pkg, k+"/"); ok {
				pkg = v + "/" + reducePkgElements(strings.Split(s, "/"))
			}
		}
	}

	ret := pkg + "." + obj
	functionCache[function] = ret
	return ret
}

// init populates moduleCache with main module and dependency package paths from build info.
func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		moduleCache[info.Main.Path] = reducePkgElements(strings.Split(info.Main.Path, "/"))
		for _, dep := range info.Deps {
			moduleCache[dep.Path] = reducePkgElements(strings.Split(dep.Path, "/"))
		}
	}
}
