package utils

import (
	"github.com/threatwinds/go-sdk/catcher"
	"reflect"
	"time"
)

func ChannelStatus[T any](channel chan T) {
	for {
		catcher.Info("channel status", map[string]any{"channel": reflect.TypeOf(channel).String(), "length": len(channel), "cap": cap(channel)})
		time.Sleep(60 * time.Second)
	}
}
