package forwarder

import (
    "code.google.com/p/go.crypto/ssh"
    "fmt"
    "io"
    "log"
    "net"
)

type SSHConfig struct {
    Username   string
    Password   string
    SSHServer  string
    RemotePort int
    LocalPort  int
}

func (c SSHConfig) RunForwarder() error {
    config := &ssh.ClientConfig{
        User: c.Username,
        Auth: []ssh.AuthMethod{
            ssh.Password(c.Password),
        },
    }

    // Dial to remote ssh server.
    conn, err := ssh.Dial("tcp", c.SSHServer, config)
    if err != nil {
        return err
    }
    defer conn.Close()

    // Request the remote side to open port
    remoteListener, err := conn.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", c.RemotePort))
    if err != nil {
        return err
    }
    defer remoteListener.Close()

    for {
        // Listen for remote connection
        r, err := remoteListener.Accept()
        if err != nil {
            log.Fatalf("listen.Accept failed: %v", err)
        }
        go func() {
            // Connect to local port
            l, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", c.LocalPort))
            if err != nil {
                log.Fatalf("unable to register tcp forward: %v", err)
            }

            //copy local writer to remote reader
            go func() {
                _, err = io.Copy(l, r)
                if err != nil {
                    log.Fatalf("io.Copy failed: %v", err)
                }
            }()

            //copy remote writer to local reader
            go func() {
                _, err = io.Copy(r, l)
                if err != nil {
                    log.Fatalf("io.Copy failed: %v", err)
                }
            }()
        }()
    }
    return nil
}
