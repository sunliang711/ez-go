package eztools

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForSignal 等待信号 SIGINT 或 SIGTERM
func WaitForSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
