package registrar

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternStart(t *testing.T) {
	t.Run("Test clear", func(t *testing.T) {
		f := &patternFormatter{}
		f.current = "<current pattern>"
		f.buffer = "<buffer value>"
		f.clear()
		assert.Equal(t, "", f.current)
		assert.Equal(t, "", f.buffer)
	})

	t.Run("Test cleanPath", func(t *testing.T) {
		f := &patternFormatter{}

		t.Run("prefixes with /", func(t *testing.T) {
			assert.Equal(t, "/api", f.cleanPath("api"))
		})
		t.Run("trims to one /", func(t *testing.T) {
			assert.Equal(t, "/api", f.cleanPath("//api"))
			assert.Equal(t, "/api", f.cleanPath("/////api"))
		})
	})

	t.Run("Test base", func(t *testing.T) {
		api := &patternFormatter{basePath: "/api"}
		apiSame := &patternFormatter{basePath: "api"}

		t.Run("cleans basePath", func(t *testing.T) {
			assert.Equal(t, "/api", api.base())
			assert.Equal(t, "/api", apiSame.base())
			assert.Equal(t, api.base(), apiSame.base())
		})
	})

	t.Run("Test set", func(t *testing.T) {
		t.Run("panics without basePath", func(t *testing.T) {
			f := &patternFormatter{}
			assert.Panics(t, func() { f.set("METHOD ") })
		})

		t.Run("panics with empty method", func(t *testing.T) {
			f := &patternFormatter{basePath: "api"}
			assert.Panics(t, func() { f.set("") })
		})

		t.Run("method uppercased, space trimmed", func(t *testing.T) {
			f := &patternFormatter{basePath: "api"}
			f.set(" mEtHoD  ")
			assert.True(t, strings.HasPrefix(f.buffer, "METHOD"))
			assert.True(t, strings.HasPrefix(f.buffer, "METHOD "))
			assert.False(t, strings.HasPrefix(f.buffer, "METHOD  "))
		})

		t.Run("buffer is '<method> <base>', current empty", func(t *testing.T) {
			f := &patternFormatter{basePath: "api"}
			f.set("mEtHoD")
			assert.Equal(t, "METHOD /api", f.buffer)
			assert.Equal(t, "", f.current)
		})
	})

	t.Run("Test method starters", func(t *testing.T) {
		api := &patternFormatter{basePath: "api"}
		methods := []func() pathSelector{
			api.Post, api.Get, api.Put, api.Patch, api.Delete,
		}

		for _, method := range methods {
			method()
			assert.True(t, api.current == "")
			assert.True(t, api.buffer != "")
			assert.False(t, strings.HasPrefix(api.buffer, " "))
			assert.True(t, strings.HasSuffix(api.buffer, " "+api.base()))
		}
	})
}

func TestPatternBuild(t *testing.T) {
	t.Run("Test Add", func(t *testing.T) {
		api := &patternFormatter{basePath: "api"}
		api.Get()

		t.Run("pattern start in buffer only", func(t *testing.T) {
			assert.Equal(t, "GET /api", api.buffer)
			assert.Equal(t, "", api.current)
		})

		api.Add("Addpath")

		t.Run("pattern start Added, Add() val in buffer", func(t *testing.T) {
			assert.Equal(t, "GET /api", api.current)
			assert.Equal(t, "/Addpath", api.buffer)
		})

		api.Add("")

		t.Run("empty Add appends, but buffers nothing", func(t *testing.T) {
			assert.Equal(t, "GET /api/Addpath", api.current)
			assert.Equal(t, "", api.buffer)
		})
	})

	t.Run("Test single", func(t *testing.T) {
		t.Run("Test single", func(t *testing.T) {
			f := &patternFormatter{}
			expected := "/budgets/{budget_id}"

			t.Run("returns formatted resource path", func(t *testing.T) {
				s := f.single("budgets", "budget")
				assert.Equal(t, expected, s)
			})
		})

		t.Run("Test resource wrappers", func(t *testing.T) {
			api := &patternFormatter{basePath: "api"}

			wrappers := []func() pathSelector{
				api.Budget, api.Account, api.Group, api.Category, api.Payee, api.Transaction, api.Member, api.Month,
			}

			for _, wrapper := range wrappers {
				api.Get()
				wrapper()
				expected := `^/[^/]+/\{[^/]+_id\}$`
				assert.Regexp(t, expected, api.buffer)
				api.clear()
			}
		})

		t.Run("Test col", func(t *testing.T) {
			t.Run("panics with empty buffer", func(t *testing.T) {
				api := &patternFormatter{basePath: "api"}
				assert.Panics(t, func() {
					api.Get().Budget().Add("").Col()
				})
			})

			api := &patternFormatter{basePath: "api"}
			api.Get()
			collection := "/budgets"

			t.Run("truncates resource to its collection path", func(t *testing.T) {
				api.Budget().Col()
				unexpected := `^/[^/]+/\{[^/]+_id\}$`
				assert.NotRegexp(t, unexpected, api.buffer)
				assert.Equal(t, collection, api.buffer)
			})
		})
	})
}

func TestPatternEnd(t *testing.T) {
	t.Run("Test end", func(t *testing.T) {
		t.Run("panics with empty pattern", func(t *testing.T) {
			api := &patternFormatter{basePath: "api"}
			assert.Panics(t, func() {
				api.end()
			})
		})

		expected := "GET /api/budgets/{budget_id}/transactions/{transaction_id}/details"

		api := &patternFormatter{basePath: "api"}
		result := api.Get().Budget().Transaction().Add("details").end()

		t.Run("returns expected pattern", func(t *testing.T) {
			assert.Equal(t, expected, result)
		})

		t.Run("formatter internals are cleared (buffer, current)", func(t *testing.T) {
			assert.Equal(t, "", api.current)
			assert.Equal(t, "", api.buffer)
		})
	})
}

func TestNewBuilder(t *testing.T) {
	t.Run("empty basePath returns error", func(t *testing.T) {
		_, err := NewBuilder("")
		assert.Error(t, err)
	})
}

func TestNewRegistrar(t *testing.T) {
	t.Run("nil mux returns error", func(t *testing.T) {
		_, err := NewRegistrar(nil)
		assert.Error(t, err)
	})
}
