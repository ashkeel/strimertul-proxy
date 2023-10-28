package strimertul_proxy

func getChannel(path string, base string) string {
	return path[len(base):]
}
