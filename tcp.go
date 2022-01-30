package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var tcpDeathChan = make(chan struct{})

func tcpProcess(conn net.Conn) {
	defer conn.Close()
	token, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Fprint(conn, "server-side error")
		infl.Println("[ERROR] reading token", err)
		return
	}
	token = strings.TrimSpace(token)
	if len(token) == 0 {
		fmt.Fprint(conn, "empty token")
		return
	} else if !isValidUUID(token) {
		fmt.Fprint(conn, "invalid token")
		return
	} else if is, err := isInUsers(token); err != nil {
		fmt.Fprint(conn, "server-side error")
		infl.Println("[ERROR] counting users", err)
		return
	} else if !is {
		fmt.Fprint(conn, "token not found")
		return
	}
	if _, ok := conns[token]; ok {
		fmt.Fprint(conn, "you already have connection; destroy it using go_offline method")
		return
	}
	conns[token] = conn
	clch := make(chan struct{})
	go func(conn net.Conn, closed chan struct{}) {
		defer func() { closed <- struct{}{} }()
		in := bufio.NewScanner(conn)
		out := bufio.NewWriter(conn)
		for in.Scan() {
			// Won't check err because won't do anything with err
			if in.Err() != nil {
				errl.Println(in.Err())
				break
			}
			switch in.Text() {
			case "ping":
				out.WriteString("pong")
			case "disconnect":
				conn.Close()
				return
			default:
				out.WriteString("don't understand")
			}
		}
	}(conn, clch)
	fmt.Fprint(conn, "success")
WAITER:
	for {
		select {
		case <-tcpDeathChan:
			break WAITER
		case <-clch:
			break WAITER
		}
	}
	delete(conns, token)
}

func listenPort(p uint16) error {
	port := ":" + strconv.Itoa(int(p))
	defer func() { tcpDeathChan <- struct{}{} }()
	ln, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			infl.Println("[ERROR] failed accepting of connection", err)
			continue
		}
		go tcpProcess(conn)
	}
}
