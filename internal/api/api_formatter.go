package api

import (
	"fmt"
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

func (ef *endpointFormatter) set(s string) *endpointFormatter {
	if ef.basePath == "" {
		panic("formatter directed to begin writing endpoint pattern, but basePath was not set")
	}
	ef.buffer = ""
	ef.current = ""
	if s != "" {
		ef.buffer = strings.TrimLeft(s, "/") + ef.base()
	}
	return ef
}

func (ef *endpointFormatter) add(s string) *endpointFormatter {
	ef.current += ef.buffer
	if s != "" {
		ef.buffer = ef.cleanPath(s)
	}
	return ef
}

func (ef *endpointFormatter) single(plural, singular string) string {
	return fmt.Sprintf("/%s/{%s_id}", plural, singular)
}

func (ef *endpointFormatter) col() *endpointFormatter {
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
	ef.current = ""
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
