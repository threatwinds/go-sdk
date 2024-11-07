package go_sdk

import "runtime"

func init() {
	notificationsChannel = make(chan *Message, runtime.NumCPU()*100)
}
