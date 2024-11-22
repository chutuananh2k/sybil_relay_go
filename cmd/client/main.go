package main

import (
	"flag"
	"log"
	"net"
	"os"
	"time"
)

var serverAddr string

func main() {
	flag.StringVar(&serverAddr, "s", "", "server address")
	message := flag.String("m", "ping", "message")
	flag.Parse()

	tcpAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
	if err != nil {
		log.Println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Println("Dial failed:", err.Error())
		os.Exit(1)
	}

	go func() {
		_, err = conn.Write([]byte(*message))
		if err != nil {
			log.Println("WRITE 1 failed:", err.Error())
			os.Exit(1)
		}

		time.Sleep(3 * time.Second)

		log.Println("WRITE 2")
		_, err = conn.Write([]byte(*message))
		if err != nil {
			log.Println("WRITE 2 failed:", err.Error())
			os.Exit(1)
		}
	}()

	go func() {
		reply := make([]byte, 1024)

		_, err = conn.Read(reply)
		if err != nil {
			log.Println("Read failed:", err.Error())
			os.Exit(1)
		}
		log.Println("reply from server=", string(reply))
	}()

	time.Sleep(10 * time.Second)
	conn.Close()
}
