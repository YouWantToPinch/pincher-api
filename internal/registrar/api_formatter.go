package registrar

import (
	"fmt"
	"regexp"
	"strings"
)

// NewBuilder returns a new endpoint pattern builder, whose
// internal formatter is set with the provided string as its base path.
func NewBuilder(basePath string) (*patternBuilder, error) {
	if basePath == "" {
		return nil, fmt.Errorf("could not create new endpoint pattern builder; no basePath provided")
	}
	return &patternBuilder{
		formatter: &patternFormatter{basePath: basePath},
	}, nil
}

type patternBuilder struct {
	formatter *patternFormatter
}

func (eb *patternBuilder) Build() methodSelector {
	return eb.formatter
}

type patternFormatter struct {
	basePath string
	buffer   string
	current  string
}

type methodSelector interface {
	Post() pathSelector
	Get() pathSelector
	Put() pathSelector
	Patch() pathSelector
	Delete() pathSelector
}

type pathSelector interface {
	Budget() pathSelector
	Account() pathSelector
	Group() pathSelector
	Category() pathSelector
	Payee() pathSelector
	Transaction() pathSelector
	Member() pathSelector
	Month() pathSelector
	Col() truncatedPathSelector
	truncatedPathSelector
}

type truncatedPathSelector interface {
	Add(string) pathSelector
	end() string
}

func (ef *patternFormatter) cleanPath(s string) string {
	return "/" + strings.TrimLeft(s, "/")
}

func (ef *patternFormatter) base() string {
	return ef.cleanPath(ef.basePath)
}

func (ef *patternFormatter) clear() {
	ef.buffer = ""
	ef.current = ""
}

func (ef *patternFormatter) set(method string) *patternFormatter {
	if ef.basePath == "" {
		panic("formatter directed to begin writing endpoint pattern, but basePath was not set")
	}
	ef.clear()
	if method != "" {
		method = strings.ToUpper((strings.TrimSpace(method))) + " "
		ef.buffer = strings.TrimLeft(method, "/") + ef.base()
	} else {
		panic("formatter directed to begin writing endpoint pattern, but method was empty")
	}
	return ef
}

func (ef *patternFormatter) Add(s string) pathSelector {
	ef.current += ef.buffer
	if s != "" {
		ef.buffer = ef.cleanPath(s)
	} else {
		ef.buffer = ""
	}
	return ef
}

func (ef *patternFormatter) single(plural, singular string) string {
	return fmt.Sprintf("/%s/{%s_id}", plural, singular)
}

func (ef *patternFormatter) Col() truncatedPathSelector {
	if ef.buffer == "" {
		panic("formatter directed to truncate buffered resource to its collection path, but buffer was empty")
	}
	r := regexp.MustCompile(`^/[^/]+/\{[^/]+_id\}$`)
	if !r.MatchString(ef.buffer) {
		panic("formatter directed to truncate buffered resource to its collection path, but found no such pattern in buffer")
	}

	if i := strings.LastIndex(ef.buffer, "/"); i >= 0 {
		ef.buffer = ef.buffer[:i]
	}
	return ef
}

func (ef *patternFormatter) end() string {
	if ef.current == "" {
		panic("formatter directed to end and return pattern, but it was found empty")
	}
	ef.Add("")
	result := ef.current
	ef.clear()
	return result
}

func (ef *patternFormatter) Get() pathSelector {
	ef.set("GET ")
	return ef
}

func (ef *patternFormatter) Post() pathSelector {
	ef.set("POST ")
	return ef
}

func (ef *patternFormatter) Put() pathSelector {
	ef.set("PUT ")
	return ef
}

func (ef *patternFormatter) Patch() pathSelector {
	ef.set("PATCH ")
	return ef
}

func (ef *patternFormatter) Delete() pathSelector {
	ef.set("DELETE ")
	return ef
}

func (ef *patternFormatter) Budget() pathSelector {
	ef.Add(ef.single("budgets", "budget"))
	return ef
}

func (ef *patternFormatter) Member() pathSelector {
	ef.Add(ef.single("members", "user"))
	return ef
}

func (ef *patternFormatter) Group() pathSelector {
	ef.Add(ef.single("groups", "group"))
	return ef
}

func (ef *patternFormatter) Category() pathSelector {
	ef.Add(ef.single("categories", "category"))
	return ef
}

func (ef *patternFormatter) Payee() pathSelector {
	ef.Add(ef.single("payees", "payee"))
	return ef
}

func (ef *patternFormatter) Account() pathSelector {
	ef.Add(ef.single("accounts", "account"))
	return ef
}

func (ef *patternFormatter) Transaction() pathSelector {
	ef.Add(ef.single("transactions", "transaction"))
	return ef
}

func (ef *patternFormatter) Month() pathSelector {
	ef.Add(ef.single("months", "month"))
	return ef
}
