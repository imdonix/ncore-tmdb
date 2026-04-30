package service

import (
	"fmt"
	"log"
	"os"
)

var ncore_host string
var ncore_user string
var ncore_pass string

func InitNCore() {

	ncore_host := os.Getenv("NCORE_HOST")
	if ncore_host == "" {
		log.Panic("missing required env vars: NCORE_HOST, NCORE_USER, NCORE_PASS")
	}

	ncore_user := os.Getenv("NCORE_USER")
	if ncore_user == "" {
		log.Panic("missing required env vars: NCORE_HOST, NCORE_USER, NCORE_PASS")
	}

	ncore_pass := os.Getenv("NCORE_PASS")
	if ncore_pass == "" {
		log.Panic("missing required env vars: NCORE_HOST, NCORE_USER, NCORE_PASS")
	}

}

func GetTorrentsNCore() string {
	return fmt.Sprintf("called ncore at %s with user %s", ncore_host, ncore_user)
}
