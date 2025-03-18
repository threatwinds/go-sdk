package utils

import (
	"os"
	"path"
)

func MkdirJoin(f ...string) (string, error) {
	address := path.Join(f...)
	err := os.MkdirAll(address, 0755)
	return address, err
}

func FileJoin(f ...string) string {
	return path.Join(f...)
}
