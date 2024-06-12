package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

func main() {
	local := "0.0.0.0:54321"

	localAddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(local): ", err)
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	dataCache := cache.New(5*time.Minute, 10*time.Minute)
	buf := make([]byte, 4096)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}

		msg := string(buf[:n])
		log.Print("Receive from ", remoteAddr, " to ", conn.LocalAddr(), " : ", msg)

		col := strings.Split(msg, " ")
		if len(col) < 1 {
			continue
		}

		switch col[0] {
		case "GET":
			item, found := dataCache.Get(col[1])
			if found {
				msg = fmt.Sprintf("OK %s", item.(string))
			} else {
				msg = "NF "
			}
		case "REG":
			remote := remoteAddr.String()
			dataCache.Set(col[1], remote, cache.NoExpiration)
			msg = fmt.Sprintf("OK %s", remote)
		default:
			msg = "NG "
		}

		_, err = conn.WriteToUDP([]byte(msg), remoteAddr)
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Send to ", remoteAddr, " from ", conn.LocalAddr().String(), " : ", msg)
	}
}
