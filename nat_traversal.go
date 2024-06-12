package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	host := "0.0.0.0:54321"
	localName := "connector"
	remoteName := "edge"
	broker := "prgmr.nohohon.jp:54321"

	hostAddr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(host): ", err)
		os.Exit(1)
	}

	brokerAddr, err := net.ResolveUDPAddr("udp", broker)
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(broker): ", err)
		os.Exit(3)
	}

	conn, err := net.ListenUDP("udp", hostAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	err = regLocal(conn, localName, brokerAddr)
	if err != nil {
		log.Fatal("regLocal(", localName, "): ", err)
		os.Exit(2)
	}

	time.Sleep(10 * time.Second)

	remoteAddr, err := getRemote(conn, remoteName, brokerAddr)
	if err != nil {
		log.Fatal("regLocal(", remoteName, "): ", err)
		os.Exit(2)
	}

	go server(conn)
	go client(conn, remoteAddr)

	for {
		time.Sleep(10 * time.Second)
	}
}

func server(conn *net.UDPConn) {
	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Receive from ", remoteAddr, " to ", conn.LocalAddr(), " : ", string(buf[:n]))
	}
}

func client(conn *net.UDPConn, remoteAddr *net.UDPAddr) {
	n := 0
	for {
		msg := fmt.Sprintf("count %d", n)
		_, err := conn.WriteToUDP([]byte(msg), remoteAddr)
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Send to ", remoteAddr, " from ", conn.LocalAddr().String(), " : ", msg)

		n++

		time.Sleep(5 * time.Second)
	}
}

func regLocal(conn *net.UDPConn, name string, brokerAddr *net.UDPAddr) error {
	msg := fmt.Sprintf("REG %s %s", name, conn.LocalAddr().String())
	_, err := conn.WriteToUDP([]byte(msg), brokerAddr)
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 4096)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		log.Fatal(err)
	}

	col := strings.Split(string(buf[:n]), " ")
	if col[0] != "OK" {
		log.Fatal("REG failed : ", name)
	}

	log.Print("REG success : ", name, " as ", col[1])

	return nil
}

func getRemote(conn *net.UDPConn, name string, brokerAddr *net.UDPAddr) (*net.UDPAddr, error) {
	msg := fmt.Sprintf("GET %s", name)
	_, err := conn.WriteToUDP([]byte(msg), brokerAddr)
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 4096)
	n, remoteAddr, err := conn.ReadFromUDP(buf)
	if err != nil {
		log.Fatal(err)
	}

	col := strings.Split(string(buf[:n]), " ")
	if col[0] != "OK" {
		log.Fatal("GET failed : ", name)
	}

	log.Print("GET success : ", name, " as ", col[1])

	remoteAddr, err = net.ResolveUDPAddr("udp", col[1])
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(regLocal): ", err)
	}

	return remoteAddr, nil
}
