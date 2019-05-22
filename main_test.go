package main

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/test/bufconn"
	"gopkg.in/awnumar/memguard.v0"
	"gopkg.in/phayes/permbits.v0"
)

func TestMain(t *testing.T) {
	t.Run("Simulate", testSimulate)
	t.Run("ShortPayload", testShortPayload)
	t.Run("SocketPermissions", testSocketPermissions)
	t.Run("ServerReadError", testServerReadError)
	t.Run("BadBuffer", testBadBuffer)
	t.Run("BadSocketPath", testBadSocketPath)
}

// Pretend we're a client and run through the main code path
func testSimulate(t *testing.T) {
	testString := "1234testing"
	oldBuffer := bufferSize
	bufferSize = len(testString) + 1
	defer func() { bufferSize = oldBuffer }()

	go main()
	time.Sleep(1 * time.Second)

	client, err := net.Dial("unix", sockDefault)
	assert.NoError(t, err)
	client.Write([]byte("P" + testString + "\n"))
	time.Sleep(1 * time.Second)
	client.Close()
	client, err = net.Dial("unix", sockDefault)
	assert.NoError(t, err)
	client.Write([]byte("G\n"))
	payload, err := ioutil.ReadAll(client)
	assert.NoError(t, err)

	quit <- syscall.SIGINT
	time.Sleep(1 * time.Second)
	assert.Equal(t, testString, string(payload))
}

// Test a short payload
func testShortPayload(t *testing.T) {
	payloadTest := "\n"
	server := bufconn.Listen(bufferSize + cmdSize + 2)
	defer server.Close()
	buf, _ := memguard.NewImmutable(bufferSize)
	defer memguard.DestroyAll()
	errChan := make(chan error, 1)
	go func(server *bufconn.Listener, buf *memguard.LockedBuffer) {
		conn, _ := server.Accept()
		errChan <- clientHandler(conn, buf)
	}(server, buf)

	client, err := server.Dial()
	defer client.Close()
	assert.NoError(t, err)
	client.Write([]byte(payloadTest))
	data, err := ioutil.ReadAll(client)
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, data)

	clientErr := <-errChan
	if clientErr == nil {
		clientErr = errors.New("test failed")
	}
	assert.Contains(t, clientErr.Error(), "Bad payload length: "+strconv.Itoa(len(payloadTest)))
}

// Test socket creation
func testSocketPermissions(t *testing.T) {
	go main()
	time.Sleep(time.Second * 1)
	f, err := os.Stat(sockDefault)
	assert.NoError(t, err)
	permissions := permbits.FileMode(f.Mode())
	if permissions != 0600 {
		t.Logf("socket file mode incorrect: %s", permissions)
		t.Fail()
	}
	quit <- syscall.SIGINT
	time.Sleep(time.Second * 1)
	_, err = os.Stat(sockDefault)
	assert.EqualError(t, err, "stat "+sockDefault+": no such file or directory")
}

// Test Listen/Accept and a read error
func testServerReadError(t *testing.T) {
	server := bufconn.Listen(bufferSize + cmdSize + 2)
	defer server.Close()
	buf, _ := memguard.NewImmutable(bufferSize)
	defer memguard.DestroyAll()
	errChan := make(chan error, 1)
	go func(server *bufconn.Listener, buf *memguard.LockedBuffer) {
		conn, _ := server.Accept()
		errChan <- clientHandler(conn, buf)
	}(server, buf)

	client, err := server.Dial()
	assert.NoError(t, err)
	client.Write([]byte("G"))
	client.Close()

	clientErr := <-errChan
	assert.Regexp(t, regexp.MustCompile(`Received error on read: EOF`), clientErr.Error())
}

// Test invalid buffer size
func testBadBuffer(t *testing.T) {
	exiter = func(s string, c int) { panic(s) }
	oldBuffer := bufferSize
	bufferSize = -1
	defer func() { bufferSize = oldBuffer }()
	assert.PanicsWithValue(t, "memguard.ErrInvalidLength: length of buffer must be greater than zero", main)
}

// Test incorrectly specified socket path
func testBadSocketPath(t *testing.T) {
	exiter = func(s string, c int) { panic(s) }
	oldSock := sockDefault
	defer func() { sockDefault = oldSock }()
	sockDefault = "/blah"
	assert.PanicsWithValue(t, "listen unix "+sockDefault+": bind: permission denied", main)
}
