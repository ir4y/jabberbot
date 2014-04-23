package main

import (
    "./forwarder"
    "flag"
    "fmt"
    "github.com/ir4y/go-xmpp"
    "log"
    "os"
    "strings"
)

var server = flag.String("server", "localhost:5222", "server")
var username = flag.String("username", "", "username")
var password = flag.String("password", "", "password")
var debug = flag.Bool("debug", false, "debug output")
var ssh_server = flag.String("ssh_server", "", "ssh server")
var ssh_username = flag.String("ssh_username", "", "username for ssh server")
var ssh_password = flag.String("ssh_password", "", "password for ssh server")

func main() {
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "usage: example [options]\n")
        flag.PrintDefaults()
        os.Exit(2)
    }
    flag.Parse()
    if *username == "" || *password == "" {
        flag.Usage()
    }

    var talk *xmpp.Client
    var err error
    options := xmpp.Options{Host: *server,
        User:     *username,
        Password: *password,
        NoTLS:    true,
        Debug:    *debug,
        Session:  true}

    talk, err = options.NewClient()

    if err != nil {
        log.Fatal(err)
    }

    var current_forwarder forwarder.Forwarder
    for {
        chat, err := talk.Recv()
        if err != nil {
            log.Fatal(err)
        }
        switch v := chat.(type) {
        case xmpp.Chat:
            var localPort int
            var remotePort int
            c, scanf_err := fmt.Sscanf(v.Text, "connect_back %d:%d", &localPort, &remotePort)

            if strings.HasPrefix(v.Text, "echo ") {
                talk.Send(xmpp.Chat{Remote: v.Remote, Type: "chat", Text: v.Text[5:]})
            } else if scanf_err == nil && c == 2 {
                ssh_config := forwarder.SSHConfig{
                    Username:   *ssh_username,
                    Password:   *ssh_password,
                    SSHServer:  *ssh_server,
                    RemotePort: remotePort,
                    LocalPort:  localPort,
                }
                current_forwarder, err = ssh_config.RunForwarder()
                if err != nil {
                    log.Fatalf("Forwarder error: %v", err)
                }
                talk.Send(xmpp.Chat{Remote: v.Remote, Type: "chat", Text: "tunel_created"})
            } else if strings.HasPrefix(v.Text, "stop_connect_back") {
                err := current_forwarder.Stop()
                if err != nil {
                    log.Fatalf("Forwarder stop error: %v", err)
                }
                talk.Send(xmpp.Chat{Remote: v.Remote, Type: "chat", Text: "tunel_closed"})
            } else {
                fmt.Println(v.Remote, v.Text)
            }
        case xmpp.Presence:
            fmt.Println(v.From, v.Show)
        }
    }
}
