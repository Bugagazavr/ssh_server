package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/kr/pty"
	"golang.org/x/crypto/ssh"

	"C"
)

var signal = make(chan bool)

//export start
func start(caddress, clogin, cpassword, cexecutable, csshKey *C.char) {
	address := C.GoString(caddress)
	login := C.GoString(clogin)
	password := C.GoString(cpassword)
	executable := strings.Split(C.GoString(cexecutable), " ")
	sshKey := C.GoString(csshKey)

	// In the latest version of crypto/ssh (after Go 1.3), the SSH server type has been removed
	// in favour of an SSH connection type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in effect, upgrading the net.Conn
	// into an ssh.ServerConn

	config := &ssh.ServerConfig{
		//Define a function to run when a client attempts a password login
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in a production setting.
			if c.User() == login && string(pass) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
		// You may also explicitly allow anonymous client authentication, though anon bash
		// sessions may not be a wise idea
		// NoClientAuth: true,
	}

	// You can generate a keypair with 'ssh-keygen -t rsa'
	privateBytes, err := ioutil.ReadFile(sshKey)
	if err != nil {
		log.Fatal("Failed to load private key (./id_rsa)")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key")
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be accepted.
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s (%s)", address, err)
	}

	// Accept all connections
	log.Printf("Listening on %s...", address)
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Printf("Failed to handshake (%s)", err)
			continue
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		// Discard all global out-of-band Requests
		go ssh.DiscardRequests(reqs)
		// Accept all channels
		go handleChannels(chans, executable)
	}
}

//export stop
func stop() {
	signal <- true
}

func handleChannels(chans <-chan ssh.NewChannel, executable []string) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go handleChannel(newChannel, executable)
	}
}

func handleChannel(newChannel ssh.NewChannel, executable []string) {
	// Since we're handling a shell, we expect a
	// channel type of "session". The also describes
	// "x11", "direct-tcpip" and "forwarded-tcpip"
	// channel types.
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	// At this point, we have the opportunity to reject the client's
	// request for another logical connection
	connection, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel (%s)", err)
		return
	}

	// Fire up bash for this session
	// cmd := exec.Command("bash", "--rcfile", "<(echo 'PS1=\"change > \"' << echo 'PATH=$PATH:./bin')")
	var primaryExecutable string
	var secondaryExcutables []string

	if len(executable) > 1 {
		for i, v := range executable {
			if i == 0 {
				primaryExecutable = v
			} else {
				secondaryExcutables = append(secondaryExcutables, v)
			}
		}
	} else {
		primaryExecutable = executable[0]
	}
	cmd := exec.Command(primaryExecutable, secondaryExcutables...)
	// cmd := exec.Command("bash", "--rcfile")

	// Prepare teardown function
	close := func() {
		connection.Close()
		_, err := cmd.Process.Wait()
		if err != nil {
			log.Printf("Failed to exit bash (%s)", err)
		}
		log.Printf("Session closed")
	}

	// Allocate a terminal for this channel
	log.Print("Creating pty...")
	bashf, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Could not start pty (%s)", err)
		close()
		return
	}

	//pipe session to bash and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, bashf)
		once.Do(close)
	}()
	go func() {
		io.Copy(bashf, connection)
		once.Do(close)
	}()

	// Sessions have out-of-band requests such as "shell", "pty-req" and "env"
	go func() {
		for req := range requests {
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) == 0 {
					req.Reply(true, nil)
				}
			case "pty-req":
				termLen := req.Payload[3]
				w, h := parseDims(req.Payload[termLen+4:])
				SetWinsize(bashf.Fd(), w, h)
				// Responding true (OK) here will let the client
				// know we have a pty ready for input
				req.Reply(true, nil)
			case "window-change":
				w, h := parseDims(req.Payload)
				SetWinsize(bashf.Fd(), w, h)
			}
		}
	}()
}

// =======================

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

// ======================

// Winsize stores the Height and Width of a terminal.
type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16 // unused
	y      uint16 // unused
}

// SetWinsize sets the size of the given pty.
func SetWinsize(fd uintptr, w, h uint32) {
	ws := &Winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}

func main() {}
