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

// Function used to reduce a Function name (from logging)
func ReduceFunctionName(function string) string {
	if ret, ok := functionCache[function]; ok {
		return ret
	}
	functionLock.Lock()
	defer functionLock.Unlock()

	dir := filepath.Dir(function)
	base := filepath.Base(function)
	comp := strings.SplitN(base, ".", 3)
	pkg := dir + "/" + comp[0]
	obj := comp[1]
	for k, v := range moduleCache {
		if k == pkg {
			pkg = v
		} else if strings.HasPrefix(pkg, k) {
			if s, ok := strings.CutPrefix(pkg, k+"/"); ok {
				pkg = v + "/" + reducePkgElements(strings.Split(s, "/"))
			}
		}
	}

	if strings.HasPrefix(obj, "(*") {
		obj = obj[2 : len(obj)-1]
	}
	obj = strings.Replace(obj, "[...]", "", 1)

	ret := pkg + " " + obj //+ "(" + fn + ")"
	functionCache[function] = ret
	return ret
}

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		if len(info.Deps) == 0 {
			info.Deps = []*debug.Module{{Path: "github.com/BeamFoundry/rays"}}
		}
		for _, dep := range info.Deps {
			dom := reducePkgElements(strings.Split(dep.Path, "/"))
			moduleCache[dep.Path] = dom
		}
	}
}
