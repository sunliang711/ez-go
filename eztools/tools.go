package eztools

import (
	"fmt"
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
