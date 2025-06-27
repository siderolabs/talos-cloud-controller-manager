package transformer

import (
	"encoding/base64"
	"reflect"
	"regexp"
	"strings"

	sm "github.com/Masterminds/semver/v3"
)

var genericMap = map[string]interface{}{
	"default":  defaultFunc,
	"empty":    empty,
	"coalesce": coalesce,
	"ternary":  ternary,

	// String functions:
	"upper":      strings.ToUpper,
	"lower":      strings.ToLower,
	"trim":       strings.TrimSpace,
	"trimSuffix": func(a, b string) string { return strings.TrimSuffix(b, a) },
	"trimPrefix": func(a, b string) string { return strings.TrimPrefix(b, a) },

	"replace":         func(o, n, s string) string { return strings.ReplaceAll(s, o, n) },
	"regexFind":       regexFind,
	"regexFindString": regexFindString,
	"regexReplaceAll": regexReplaceAll,

	"contains":  func(substr string, str string) bool { return strings.Contains(str, substr) },
	"hasPrefix": func(substr string, str string) bool { return strings.HasPrefix(str, substr) },
	"hasSuffix": func(substr string, str string) bool { return strings.HasSuffix(str, substr) },

	// SemVer:
	"semver":        semver,
	"semverCompare": semverCompare,

	// Encoding functions:
	"b64enc": base64encode,
	"b64dec": base64decode,

	// String slice functions:
	"getValue": getValue,
}

// GenericFuncMap returns a copy of the basic function map as a map[string]interface{}.
func GenericFuncMap() map[string]interface{} {
	gfm := make(map[string]interface{}, len(genericMap))
	for k, v := range genericMap {
		gfm[k] = v
	}

	return gfm
}

// Source from https://github.com/Masterminds/sprig/blob/master/defaults.go with some modifications
//
// Checks whether `given` is set, and returns default if not set.
func defaultFunc(d any, given ...any) any {
	if empty(given) || empty(given[0]) {
		return d
	}

	return given[0]
}

// empty returns true if the given value has the zero value for its type.
func empty(given any) bool {
	g := reflect.ValueOf(given)

	return !g.IsValid() || g.IsZero()
}

// coalesce returns the first non-empty value.
func coalesce(v ...any) any {
	for _, val := range v {
		if !empty(val) {
			return val
		}
	}

	return nil
}

// ternary returns the first value if the last value is true, otherwise returns the second value.
func ternary(vt any, vf any, v bool) any {
	if v {
		return vt
	}

	return vf
}

func regexFindString(regex string, s string, n int) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	matches := r.FindStringSubmatch(s)

	if len(matches) < n+1 {
		return "", nil
	}

	return matches[n], nil
}

func regexReplaceAll(regex string, s string, repl string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	return r.ReplaceAllString(s, repl), nil
}

func regexFind(regex string, s string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	return r.FindString(s), nil
}

func semverCompare(constraint, version string) (bool, error) {
	c, err := sm.NewConstraint(constraint)
	if err != nil {
		return false, err
	}

	v, err := sm.NewVersion(version)
	if err != nil {
		return false, err
	}

	return c.Check(v), nil
}

func semver(version string) (*sm.Version, error) {
	return sm.NewVersion(version)
}

func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func base64decode(v string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getValue(source string, key string) string {
	parts := strings.Split(source, ";")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if kv[0] == key {
			return kv[1]
		}
	}

	return ""
}
