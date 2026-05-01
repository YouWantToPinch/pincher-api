package api

import (
	"fmt"
	"regexp"
	"strings"
)

type endpointFormatter struct {
	basePath string
	buffer   string
	current  string
}

func (ef *endpointFormatter) cleanPath(s string) string {
	return "/" + strings.TrimLeft(s, "/")
}

func (ef *endpointFormatter) base() string {
	return ef.cleanPath(ef.basePath)
}

func (ef *endpointFormatter) clear() {
	ef.buffer = ""
	ef.current = ""
}

func (ef *endpointFormatter) set(method string) *endpointFormatter {
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

func (ef *endpointFormatter) add(s string) *endpointFormatter {
	ef.current += ef.buffer
	if s != "" {
		ef.buffer = ef.cleanPath(s)
	} else {
		ef.buffer = ""
	}
	return ef
}

func (ef *endpointFormatter) single(plural, singular string) string {
	return fmt.Sprintf("/%s/{%s_id}", plural, singular)
}

func (ef *endpointFormatter) col() *endpointFormatter {
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

func (ef *endpointFormatter) end() string {
	if ef.current == "" {
		panic("formatter directed to end and return pattern, but it was found empty")
	}
	ef.add("")
	result := ef.current
	ef.clear()
	return result
}

func (ef *endpointFormatter) get() *endpointFormatter {
	ef.set("GET ")
	return ef
}

func (ef *endpointFormatter) post() *endpointFormatter {
	ef.set("POST ")
	return ef
}

func (ef *endpointFormatter) put() *endpointFormatter {
	ef.set("PUT ")
	return ef
}

func (ef *endpointFormatter) patch() *endpointFormatter {
	ef.set("PATCH ")
	return ef
}

func (ef *endpointFormatter) delete() *endpointFormatter {
	ef.set("DELETE ")
	return ef
}

func (ef *endpointFormatter) budget() *endpointFormatter {
	ef.add(ef.single("budgets", "budget"))
	return ef
}

func (ef *endpointFormatter) member() *endpointFormatter {
	ef.add(ef.single("members", "user"))
	return ef
}

func (ef *endpointFormatter) group() *endpointFormatter {
	ef.add(ef.single("groups", "group"))
	return ef
}

func (ef *endpointFormatter) category() *endpointFormatter {
	ef.add(ef.single("categories", "category"))
	return ef
}

func (ef *endpointFormatter) payee() *endpointFormatter {
	ef.add(ef.single("payees", "payee"))
	return ef
}

func (ef *endpointFormatter) account() *endpointFormatter {
	ef.add(ef.single("accounts", "account"))
	return ef
}

func (ef *endpointFormatter) transaction() *endpointFormatter {
	ef.add(ef.single("transactions", "transaction"))
	return ef
}

func (ef *endpointFormatter) month() *endpointFormatter {
	ef.add(ef.single("months", "month"))
	return ef
}
