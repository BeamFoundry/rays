package logger

import (
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

const defaultMaxFunctionCacheSize = 1000

var maxFunctionCacheSize = defaultMaxFunctionCacheSize

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

	bracketDepth := 0
	lastSlash := -1
scanLoop:
	for i := len(function) - 1; i >= 0; i-- {
		switch function[i] {
		case ']':
			bracketDepth++
		case '[':
			bracketDepth--
		case '/':
			if bracketDepth == 0 {
				lastSlash = i
				break scanLoop
			}
		default:
		}
	}
	dir := "."
	base := function
	if lastSlash != -1 {
		dir = function[:lastSlash]
		base = function[lastSlash+1:]
	}

	var comp []string
	if idx := strings.Index(base, ".(*"); idx != -1 {
		if end := strings.LastIndex(base, ")"); end != -1 && end > idx+2 {
			pkgPart := base[:idx]
			recv := base[idx+1 : end+1]
			method := strings.TrimPrefix(base[end+1:], ".")
			comp = []string{pkgPart, recv, method}
		}
	}
	if len(comp) == 0 {
		if bStart := strings.Index(base, "["); bStart != -1 && !strings.Contains(base, "(*") {
			bEnd := strings.LastIndex(base, "]")
			dotIdx := strings.LastIndex(base[:bStart], ".")
			if bEnd > bStart && dotIdx != -1 && dotIdx+1 < bStart {
				pkgPart := base[:dotIdx]
				recv := base[dotIdx+1 : bEnd+1]
				method := strings.TrimPrefix(base[bEnd+1:], ".")
				comp = []string{pkgPart, recv, method}
			}
		}
		if len(comp) == 0 {
			comp = strings.SplitN(base, ".", 3)
		}
	}

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

	pkg := comp[0]
	if dir != "." {
		pkg = dir + "/" + comp[0]
	}
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
	if len(comp) > 2 && comp[2] != "" && strings.Contains(obj, "[") {
		ret += "." + comp[2]
	}
	if len(functionCache) < maxFunctionCacheSize {
		functionCache[function] = ret
	}
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
