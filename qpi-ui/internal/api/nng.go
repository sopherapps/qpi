package api

import (
	"crypto/tls"
	"fmt"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol"
	_ "go.nanomsg.org/mangos/v3/transport/tlstcp"
)

// getListener gets a TLS Listener at the given port
func getListener(sock protocol.Socket, port int, tlsConfig *tls.Config) (mangos.Listener, error) {
	addr := fmt.Sprintf("tls+tcp://0.0.0.0:%d", port)
	l, err := sock.NewListener(addr, map[string]any{mangos.OptionTLSConfig: tlsConfig})
	if err != nil {
		return nil, fmt.Errorf("Listerner error: %w", err)
	}

	return l, nil
}
