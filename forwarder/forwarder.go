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

type Forwarder struct {
    ssh_connection  *ssh.Client
    remote_listener net.Listener
}

func (f *Forwarder) Stop() error {
    err := f.remote_listener.Close()
    if err != nil {
        return err
    }

    err = f.ssh_connection.Close()
    if err != nil {
        return err
    }
    return nil
}

func (c SSHConfig) RunForwarder() (Forwarder, error) {
    config := &ssh.ClientConfig{
        User: c.Username,
        Auth: []ssh.AuthMethod{
            ssh.Password(c.Password),
        },
    }

    // Dial to remote ssh server.
    conn, err := ssh.Dial("tcp", c.SSHServer, config)
    if err != nil {
        return Forwarder{}, err
    }

    // Request the remote side to open port
    remoteListener, err := conn.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", c.RemotePort))
    if err != nil {
        return Forwarder{}, err
    }

    go func() {
        for {
            // Listen for remote connection
            r, err := remoteListener.Accept()
            if err != nil {
                if err == io.EOF {
                    return
                }
                log.Fatalf("listen.Accept failed: %v", err)
            }
            go func() {
                // Connect to local port
                l, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", c.LocalPort))
                if err != nil {
                    log.Printf("unable to register tcp forward: %v", err)
                    return
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
    }()
    return Forwarder{ssh_connection: conn, remote_listener: remoteListener}, nil
}
