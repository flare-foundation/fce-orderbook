package server

import (
	"extension-scaffold/internal/typesserver"
	"extension-scaffold/pkg/decoder"
	"extension-scaffold/pkg/types"
)

// StartTypesServer creates the decoder registry, registers all decoders,
// and starts the types-server HTTP server in a goroutine.
func StartTypesServer(port int) {
	registry := decoder.NewRegistry()
	types.RegisterDecoders(registry)

	s := typesserver.New(registry)
	go s.ListenAndServe(port) //nolint:errcheck
}
