package eztools

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type SignalHandler func()

// WaitForSignal 等待信号 SIGINT 或 SIGTERM
// handler 为信号处理函数, 为 nil 时不处理
func WaitForSignal(handler SignalHandler) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	sig := <-sigCh
	fmt.Printf("Received signal: %v\n", sig)

	if handler != nil {
		handler()
	}
}

func ServeTLS(srv *http.Server, listen net.Listener, certFile, keyFile string) error {
	if certFile == "" || keyFile == "" {
		return srv.Serve(listen)
	}
	return srv.ServeTLS(listen, certFile, keyFile)
}

func ListenAndServeTLS(srv *http.Server, certFile, keyFile string) error {
	if certFile == "" || keyFile == "" {
		return srv.ListenAndServe()
	}
	return srv.ListenAndServeTLS(certFile, keyFile)
}
