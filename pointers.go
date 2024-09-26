package go_sdk

func PointerOf[t any](s t) *t {
	return &s
}
