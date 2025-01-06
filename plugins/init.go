package plugins

import (
	"runtime"
)

// init initializes the notifications channel with a buffer size based on the number of CPU cores.
// The buffer size is set to 100 times the number of CPUs to allow for efficient message handling.
func init() {
	notificationsChannel = make(chan *Message, runtime.NumCPU()*100)
}
