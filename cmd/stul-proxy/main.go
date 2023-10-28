package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	strimertul_proxy "github.com/ashkeel/strimertul-proxy"
)

func main() {
	bind := ":8000"
	bindEnv := os.Getenv("BIND")
	if bindEnv != "" {
		bind = bindEnv
	}

	authKeys := parseKeys(os.Getenv("AUTH"))
	if len(authKeys) == 0 {
		log.Fatal("no channels configured, make sure AUTH is set")
	}

	proxy := strimertul_proxy.NewProxy(authKeys)
	log.Fatal(http.ListenAndServe(bind, proxy))
}

func parseKeys(keys string) map[string]string {
	parts := strings.Split(keys, ",")
	authKeys := make(map[string]string)
	for _, part := range parts {
		if part == "" {
			continue
		}
		kv := strings.Split(part, ":")
		channel := strings.TrimSpace(kv[0])
		key := strings.TrimSpace(kv[1])
		authKeys[channel] = key
		log.Printf("added channel: %s", channel)
	}
	return authKeys
}
