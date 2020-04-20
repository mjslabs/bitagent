package server

import (
	"log"

	"github.com/awnumar/memguard"
)

func errorAndSafeExit(s string, c int) {
	log.Println(s)
	memguard.SafeExit(c)
}
