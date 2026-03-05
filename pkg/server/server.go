package server

import extension "extension-scaffold/internal/extension"

// StartExtension creates and starts the template extension server in a goroutine.
func StartExtension(extensionPort, signPort int) {
	e := extension.New(extensionPort, signPort)
	go e.Server.ListenAndServe() //nolint:errcheck
}
