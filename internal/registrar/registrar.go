// Package registrar provides types for building out endpoint patterns in a declarative manner,
// as well as the namesake 'registrar' type, used to perform final validation steps on the pattern.
//
// The types provide robust internal validation, with each function used to build a pattern potentially
// narrowing scope for developers via interfaces. This keeps endpoints readable, consistent, and free
// of typos.
package registrar
