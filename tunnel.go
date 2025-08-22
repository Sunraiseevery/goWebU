package main

import (
	"fmt"
	"io"
	"net"
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

func (t *Tunnel) Start(host string, port int, user string, auth ssh.AuthMethod, lport int, rhost string, rport int) error {
	conf := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), conf)
	if err != nil {
		return err
	}
	t.client = c
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", lport))
	if err != nil {
		c.Close()
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
	rconn, err := t.client.Dial("tcp", fmt.Sprintf("%s:%d", rhost, rport))
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

func (tm *TunnelManager) Start(sessionID string, host string, port int, user string, auth ssh.AuthMethod, lport int, rhost string, rport int, onEvent func(level, msg string)) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, ok := tm.m[sessionID]; ok {
		return fmt.Errorf("session exists")
	}
	t := NewTunnel(onEvent)
	if err := t.Start(host, port, user, auth, lport, rhost, rport); err != nil {
		return err
	}
	tm.m[sessionID] = t
	if onEvent != nil {
		onEvent("info", fmt.Sprintf("tunnel started 127.0.0.1:%d -> %s:%d", lport, rhost, rport))
	}
	return nil
}

func (tm *TunnelManager) Stop(sessionID string) {
	tm.mu.Lock()
	t := tm.m[sessionID]
	delete(tm.m, sessionID)
	tm.mu.Unlock()
	if t != nil {
		t.Stop()
	}
}
