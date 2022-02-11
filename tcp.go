package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var tcpDeathChan = make(chan struct{})

func tcpProcess(conn net.Conn) {
	defer conn.Close()
	token, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Fprint(conn, "server-side error\n")
		infl.Println("[ERROR] reading token", err)
		return
	}
	token = strings.TrimSpace(token)
	if len(token) == 0 {
		fmt.Fprint(conn, "empty token\n")
		return
	} else if !isValidUUID(token) {
		fmt.Fprint(conn, "invalid token\n")
		return
	} else if is, err := isInUsers(token); err != nil {
		fmt.Fprint(conn, "server-side error\n")
		infl.Println("[ERROR] counting users", err)
		return
	} else if !is {
		fmt.Fprint(conn, "token not found")
		return
	}
	if _, ok := conns[token]; ok {
		fmt.Fprint(conn, "you already have connection; destroy it using go_offline method\n")
		return
	}
	conns[token] = cConn{
		Conn: conn,
		last: time.Now(),
	}
	fmt.Fprint(conn, "success\n")
WAITER:
	for {
		cc, ok := conns[token]
		if !ok {
			break WAITER
		} else if _, err := conn.Read([]byte{}); err != nil {
			break WAITER
		} else if time.Now().Sub(cc.last) <= 2*time.Minute {
			break WAITER
		}
		select {
		case <-tcpDeathChan:
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
