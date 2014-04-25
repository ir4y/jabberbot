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
    Debug      bool
}

type Forwarder struct {
    ssh_connection  *ssh.Client
    remote_listener *net.Listener
}

func (f *Forwarder) Close() error {
    err := (*f.remote_listener).Close()
    if err != nil {
        return err
    }

    err = f.ssh_connection.Close()
    if err != nil {
        return err
    }
    return nil
}

func (c SSHConfig) RunForwarder() (*Forwarder, error) {
    config := &ssh.ClientConfig{
        User: c.Username,
        Auth: []ssh.AuthMethod{
            ssh.Password(c.Password),
        },
    }

    // Dial to remote ssh server.
    conn, err := ssh.Dial("tcp", c.SSHServer, config)
    if err != nil {
        return nil, err
    }

    // Request the remote side to open port
    remoteListener, err := conn.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", c.RemotePort))
    if err != nil {
        return nil, err
    }

    go func() {
        if c.Debug {
            log.Println("Start Listner loop")
            defer log.Println("Exit Listner loop")
        }
        defer remoteListener.Close()
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
                    if c.Debug {
                        defer log.Println("Close l-writer to r-reader stream")
                    }
                    io.Copy(l, r)
                    defer l.Close()
                }()

                //copy remote writer to local reader
                go func() {
                    if c.Debug {
                        defer log.Println("Close r-writer to l-reader stream")
                    }
                    io.Copy(r, l)
                }()
            }()
        }
    }()
    return &Forwarder{ssh_connection: conn,
        remote_listener: &remoteListener}, nil

}
