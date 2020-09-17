package cmd

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/awnumar/memguard"
	"github.com/mitchellh/go-homedir"
)

// Size of the secret that'll be stored for the client
var bufferSize = 256

// Size of commands the agent understands
var cmdSize = 1

// Channel for sending signals to (for graceful shutdown)
var quit = make(chan os.Signal)

// Default path to the unix socket, if not otherwise specified on the cmd line
var sockDefault string

// How to die on an unrecoverable error
var exiter = errorAndSafeExit

func init() {
	// Try to default the socket to the caller's home
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalln("can't figure out where to put our socket:", err)
	}
	sockDefault = filepath.Join(home, ".bitagent.sock")
}

// Server -
func Server() {
	sockAddr := sockDefault
	if len(os.Args) == 2 && string(os.Args[1][0]) == "/" {
		sockAddr = os.Args[1]
	}
	syscall.Umask(0177)
	defer os.RemoveAll(sockAddr)

	memguard.DisableUnixCoreDumps()
	buf, err := memguard.NewImmutable(bufferSize)
	if err != nil {
		exiter(err.Error(), 1)
	}
	defer memguard.DestroyAll()

	l, err := net.Listen("unix", sockAddr)
	if err != nil {
		exiter(err.Error(), 1)
	}
	defer l.Close()

	// Accept loop
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("exiting accept loop:", err)
				return
			}
			go func() {
				err := clientHandler(conn, buf)
				if err != nil {
					log.Println(err)
				}
			}()
		}
	}()

	// Handle graceful shutdown
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}

func clientHandler(c net.Conn, secret *memguard.LockedBuffer) error {
	defer c.Close()

	// Load the client request and do a sanity check
	payload, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		return errors.New("Received error on read: " + err.Error())
	}
	if len(payload) < cmdSize+1 || len(payload) > cmdSize+bufferSize+1 {
		return errors.New("Bad payload length: " + strconv.Itoa(len(payload)))
	}

	switch string(payload[0:cmdSize]) {
	case "G":
		if secret.Buffer()[0] != 0 {
			io.WriteString(c, strings.Trim(string(secret.Buffer()), "\n\x00"))
		}
	case "P":
		// Save secret to buffer, being very picky about error conditions
		if err := secret.MakeMutable(); err != nil {
			exiter("Error making locked buffer mutable: "+err.Error(), 1)
		}
		if err := secret.Wipe(); err != nil {
			exiter("Error wiping locked buffer: "+err.Error(), 1)
		}
		if err := secret.Move([]byte(payload[cmdSize:])); err != nil {
			exiter("Error updating secret: "+err.Error(), 1)
		}
		if err := secret.MakeImmutable(); err != nil {
			exiter("Error making locked buffer immutable: "+err.Error(), 1)
		}
	}
	return nil
}

func errorAndSafeExit(s string, c int) {
	log.Println(s)
	memguard.SafeExit(c)
}
