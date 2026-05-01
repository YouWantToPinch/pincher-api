package registrar

import (
	"fmt"
	"log/slog"
	"net/http"
)

func NewRegistrar(mux *http.ServeMux) (*Registrar, error) {
	if mux == nil {
		return nil, fmt.Errorf("could not create new registrar; mux not provided")
	}
	return &Registrar{
		registry: map[string]struct{}{},
		mux:      mux,
	}, nil
}

type Registrar struct {
	registry map[string]struct{} // patterns already registered as endpoints
	mux      *http.ServeMux      // request multiplexer to register inputs to
}

func (r *Registrar) register(pattern string, fn http.HandlerFunc) {
	if _, ok := r.registry[pattern]; ok {
		panic(fmt.Sprintf("handler already registered with pattern: %s", pattern))
	}
	r.registry[pattern] = struct{}{}
	r.mux.HandleFunc(pattern, fn)
	slog.Info("Registered handler in mux with pattern: " + pattern)
}

func (r *Registrar) validate(ef *patternFormatter) string {
	if ef == nil {
		panic("could not register API handler; nil formatter provided")
	} else if ef.current == "" {
		panic("could not register API handler; no pattern provided")
	} else if r.mux == nil {
		panic("could not register API handler; mux was nil")
	}

	pattern := ef.end()
	return pattern
}

func (r *Registrar) Handle(ef truncatedPathSelector, fn http.HandlerFunc) {
	assert, ok := ef.(*patternFormatter)
	if !ok {
		panic("could not assert input interface as *patternFormatter")
	}
	r.register(r.validate(assert), fn)
}
