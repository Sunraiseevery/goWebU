package app

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Tunnel represents a single SSH tunnel.
type Tunnel struct {
	client   *ssh.Client
	listener net.Listener
	onEvent  func(level, msg string)
}

func NewTunnel(onEvent func(level, msg string)) *Tunnel {
	return &Tunnel{onEvent: onEvent}
}

// Connect establishes the SSH connection to the remote host.
func (t *Tunnel) Connect(host string, port int, user string, auth ssh.AuthMethod) error {
	conf := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	c, err := ssh.Dial("tcp", addr, conf)
	if err != nil {
		return err
	}
	t.client = c
	if t.onEvent != nil {
		t.onEvent("info", fmt.Sprintf("ssh connected to %s", addr))
	}
	return nil
}

// StartForward begins forwarding a local port to the remote host.
func (t *Tunnel) StartForward(lport int, rhost string, rport int) error {
	if t.client == nil {
		return fmt.Errorf("ssh connection not established")
	}
	localAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(lport))
	ln, err := listenLocal(localAddr)
	if err != nil {
		return err
	}
	t.listener = ln
	go t.acceptLoop(rhost, rport)
	return nil
}

func (t *Tunnel) acceptLoop(rhost string, rport int) {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if t.onEvent != nil {
				t.onEvent("error", err.Error())
			}
			return
		}
		go t.forward(conn, rhost, rport)
	}
}

func (t *Tunnel) forward(lconn net.Conn, rhost string, rport int) {
	defer lconn.Close()
	raddr := net.JoinHostPort(rhost, strconv.Itoa(rport))
	rconn, err := t.client.Dial("tcp", raddr)
	if err != nil {
		if t.onEvent != nil {
			t.onEvent("error", err.Error())
		}
		return
	}
	defer rconn.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(lconn, rconn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(rconn, lconn)
	}()
	wg.Wait()
}

// Alive checks if the underlying SSH connection is still active by sending
// a keepalive request. It returns false if the connection is lost.
func (t *Tunnel) Alive() bool {
	if t.client == nil {
		return false
	}
	_, _, err := t.client.SendRequest("keepalive@openssh.com", true, nil)
	return err == nil
}

// Stop shuts down any active forwarding and SSH connection.
func (t *Tunnel) Stop() {
	if t.listener != nil {
		t.listener.Close()
	}
	if t.client != nil {
		t.client.Close()
	}
}

// TunnelManager tracks sessions.
type TunnelManager struct {
	mu sync.Mutex
	m  map[string]*Tunnel
}

func NewTunnelManager() *TunnelManager {
	return &TunnelManager{m: make(map[string]*Tunnel)}
}

// Connect creates a session and establishes the SSH connection.
func (tm *TunnelManager) Connect(sessionID string, host string, port int, user string, auth ssh.AuthMethod, onEvent func(level, msg string)) error {
	tm.mu.Lock()
	if _, ok := tm.m[sessionID]; ok {
		tm.mu.Unlock()
		return fmt.Errorf("session exists")
	}
	t := NewTunnel(onEvent)
	tm.m[sessionID] = t
	tm.mu.Unlock()

	if err := t.Connect(host, port, user, auth); err != nil {
		tm.Stop(sessionID)
		return err
	}

	// Watch for connection closure and cleanup automatically.
	go func() {
		err := t.client.Conn.Wait()
		if onEvent != nil {
			if err != nil {
				onEvent("error", err.Error())
			} else {
				onEvent("info", "ssh disconnected")
			}
		}
		tm.Stop(sessionID)
	}()

	return nil
}

// Forward starts port forwarding on an existing session.
func (tm *TunnelManager) Forward(sessionID string, lport int, rhost string, rport int) error {
	tm.mu.Lock()
	t, ok := tm.m[sessionID]
	tm.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found")
	}
	if err := t.StartForward(lport, rhost, rport); err != nil {
		return err
	}
	if t.onEvent != nil {
		lAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(lport))
		rAddr := net.JoinHostPort(rhost, strconv.Itoa(rport))
		t.onEvent("info", fmt.Sprintf("tunnel started %s -> %s", lAddr, rAddr))
	}
	return nil
}

// Stop terminates a session and cleans up resources.
func (tm *TunnelManager) Stop(sessionID string) {
	tm.mu.Lock()
	t := tm.m[sessionID]
	delete(tm.m, sessionID)
	tm.mu.Unlock()
	if t != nil {
		t.Stop()
	}
}

// Alive reports whether the session's SSH connection is still active. If the
// connection has been lost, the session is removed and false is returned.
func (tm *TunnelManager) Alive(sessionID string) bool {
	tm.mu.Lock()
	t, ok := tm.m[sessionID]
	tm.mu.Unlock()
	if !ok {
		return false
	}
	if !t.Alive() {
		tm.Stop(sessionID)
		return false
	}
	return true
}
