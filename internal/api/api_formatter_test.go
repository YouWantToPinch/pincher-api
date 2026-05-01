package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternStart(t *testing.T) {
	t.Run("Test clear", func(t *testing.T) {
		f := &endpointFormatter{}
		f.current = "<current pattern>"
		f.buffer = "<buffer value>"
		f.clear()
		assert.Equal(t, "", f.current)
		assert.Equal(t, "", f.buffer)
	})

	t.Run("Test cleanPath", func(t *testing.T) {
		f := &endpointFormatter{}

		t.Run("prefixes with /", func(t *testing.T) {
			assert.Equal(t, "/api", f.cleanPath("api"))
		})
		t.Run("trims to one /", func(t *testing.T) {
			assert.Equal(t, "/api", f.cleanPath("//api"))
			assert.Equal(t, "/api", f.cleanPath("/////api"))
		})
	})

	t.Run("Test base", func(t *testing.T) {
		api := &endpointFormatter{basePath: "/api"}
		apiSame := &endpointFormatter{basePath: "api"}

		t.Run("cleans basePath", func(t *testing.T) {
			assert.Equal(t, "/api", api.base())
			assert.Equal(t, "/api", apiSame.base())
			assert.Equal(t, api.base(), apiSame.base())
		})
	})

	t.Run("Test set", func(t *testing.T) {
		t.Run("panics without basePath", func(t *testing.T) {
			f := &endpointFormatter{}
			assert.Panics(t, func() { f.set("METHOD ") })
		})

		t.Run("panics with empty method", func(t *testing.T) {
			f := &endpointFormatter{basePath: "api"}
			assert.Panics(t, func() { f.set("") })
		})

		t.Run("method uppercased, space trimmed", func(t *testing.T) {
			f := &endpointFormatter{basePath: "api"}
			f.set(" mEtHoD  ")
			assert.True(t, strings.HasPrefix(f.buffer, "METHOD"))
			assert.True(t, strings.HasPrefix(f.buffer, "METHOD "))
			assert.False(t, strings.HasPrefix(f.buffer, "METHOD  "))
		})

		t.Run("buffer is '<method> <base>', current empty", func(t *testing.T) {
			f := &endpointFormatter{basePath: "api"}
			f.set("mEtHoD")
			assert.Equal(t, "METHOD /api", f.buffer)
			assert.Equal(t, "", f.current)
		})
	})

	t.Run("Test method starters", func(t *testing.T) {
		api := &endpointFormatter{basePath: "api"}
		methods := []func() *endpointFormatter{
			api.post, api.get, api.put, api.patch, api.delete,
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
	t.Run("Test add", func(t *testing.T) {
		api := &endpointFormatter{basePath: "api"}
		api.get()

		t.Run("pattern start in buffer only", func(t *testing.T) {
			assert.Equal(t, "GET /api", api.buffer)
			assert.Equal(t, "", api.current)
		})

		api.add("addpath")

		t.Run("pattern start added, add() val in buffer", func(t *testing.T) {
			assert.Equal(t, "GET /api", api.current)
			assert.Equal(t, "/addpath", api.buffer)
		})

		api.add("")

		t.Run("empty add appends, but buffers nothing", func(t *testing.T) {
			assert.Equal(t, "GET /api/addpath", api.current)
			assert.Equal(t, "", api.buffer)
		})
	})

	t.Run("Test single", func(t *testing.T) {
		t.Run("Test single", func(t *testing.T) {
			f := &endpointFormatter{}
			expected := "/budgets/{budget_id}"

			t.Run("returns formatted resource path", func(t *testing.T) {
				s := f.single("budgets", "budget")
				assert.Equal(t, expected, s)
			})
		})

		t.Run("Test resource wrappers", func(t *testing.T) {
			api := &endpointFormatter{basePath: "api"}

			wrappers := []func() *endpointFormatter{
				api.budget, api.account, api.group, api.category, api.payee, api.transaction, api.member, api.month,
			}

			for _, wrapper := range wrappers {
				api.get()
				wrapper()
				expected := `^/[^/]+/\{[^/]+_id\}$`
				assert.Regexp(t, expected, api.buffer)
				api.clear()
			}
		})

		t.Run("Test col", func(t *testing.T) {
			t.Run("panics with empty buffer", func(t *testing.T) {
				api := &endpointFormatter{basePath: "api"}
				assert.Panics(t, func() {
					api.get().budget().add("").col()
				})
			})

			api := &endpointFormatter{basePath: "api"}
			api.get()
			collection := "/budgets"

			t.Run("truncates resource to its collection path", func(t *testing.T) {
				api.budget().col()
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
			api := &endpointFormatter{basePath: "api"}
			assert.Panics(t, func() {
				api.end()
			})
		})

		expected := "GET /api/budgets/{budget_id}/transactions/{transaction_id}/details"

		api := &endpointFormatter{basePath: "api"}
		result := api.get().budget().transaction().add("details").end()

		t.Run("returns expected pattern", func(t *testing.T) {
			assert.Equal(t, expected, result)
		})

		t.Run("formatter internals are cleared (buffer, current)", func(t *testing.T) {
			assert.Equal(t, "", api.current)
			assert.Equal(t, "", api.buffer)
		})
	})
}
