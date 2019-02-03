package http

import (
	"io"
	"net/http"
)

// MuxHandlers binds a handler function to the passed in multiplexer. By
// inverting this to set here rather than when creating the server it keeps the
// configuration closer to the handler.
func MuxHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/pulse", healthCheckHandler)
}

// healthCheckHandler is a simple http handler that writes `ok` to the
// requester.
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "ok")
}
