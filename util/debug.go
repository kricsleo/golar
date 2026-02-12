package util

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var initialized = false
var envCompiled = false
var (
	envMatches *regexp.Regexp
	envSkips *regexp.Regexp
)
func init() {
	initialized = true
	applyDebugEnv(debuggers)
}

func applyDebugEnv(debuggers []*Debug) {
	if !envCompiled {
		envCompiled = true
		debug, ok := os.LookupEnv("DEBUG")
		if !ok {
			return
		}
		envMatches, envSkips = compileDebug(debug)
	}
	applyDebugCompiled(envMatches, envSkips, debuggers)
}

func applyDebug(debug string, debuggers []*Debug) {
	matches, skips := compileDebug(debug)
	applyDebugCompiled(matches, skips, debuggers)
}

func applyDebugCompiled(matches, skips *regexp.Regexp, debuggers []*Debug) {
	if matches == nil {
		return
	}
	for _, d := range debuggers {
		if skips != nil && skips.MatchString(d.namespace) {
			continue
		}
		d.enabled = matches.MatchString(d.namespace)
	}
}

func compileDebug(debug string) (matches *regexp.Regexp, skips *regexp.Regexp) {
	if debug == "" {
		return
	}

	var (
		matchesBuilder strings.Builder
		skipsBuilder   strings.Builder
	)
	matchesBuilder.WriteByte('^')
	skipsBuilder.WriteByte('^')
	for part := range strings.SplitSeq(debug, ",") {
		if part == "" {
			continue
		}

		builder := &matchesBuilder
		if part[0] == '-' {
			part = part[1:]
			builder = &skipsBuilder
			if part == "" {
				continue
			}
		}

		if builder.Len() > 1 {
			builder.WriteByte('|')
		}
		lastIdx := 0
		for i := 0; i < len(part); i++ {
			if part[i] == '*' {
				builder.WriteString(part[lastIdx:i])
				builder.WriteString(".*")
				lastIdx = i + 1
				continue
			}
		}
		builder.WriteString(part[lastIdx:])
	}
	if matchesBuilder.Len() <= 1 {
		return
	}
	matchesBuilder.WriteByte('$')
	skipsBuilder.WriteByte('$')

	matches, _ = regexp.Compile(matchesBuilder.String())
	if skipsBuilder.Len() > 2 {
		skips, _ = regexp.Compile(skipsBuilder.String())
	}
	return
}

var debuggers []*Debug

type Debug struct {
	enabled   bool
	namespace string
}

func NewDebug(namespace string) *Debug {
	d := newDebugNotInitialized(namespace)
	if initialized {
		applyDebugEnv([]*Debug{d})
	} else {
		debuggers = append(debuggers, d)
	}
	return d
}
func newDebugNotInitialized(namespace string) *Debug {
	d := &Debug{
		enabled:   false,
		namespace: "golar:" + namespace,
	}
	return d
}

func (d Debug) Print(args ...string) {
	if d.enabled {
		os.Stderr.WriteString(d.prefix() + strings.Join(args, " ") + "\n")
	}
}

func (d Debug) Printf(format string, a ...any) {
	if d.enabled {
		fmt.Fprintf(os.Stderr, d.prefix()+format + "\n", a...)
	}
}

func (d Debug) prefix() string {
	return d.namespace + " "
}
