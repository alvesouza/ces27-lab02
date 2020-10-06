package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

type messageType int8

const (
	CSREPLY = messageType(3)
)

type processStruct struct {
	Id          int
	Clock       int
	Address     string
	TypeMessage messageType
	Texto       string
}

func CheckError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}

func main() {
	var processInfoReceived processStruct
	Address, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	CheckError(err)
	Connection, err := net.ListenUDP("udp", Address)
	CheckError(err)
	LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	CheckError(err)

	defer Connection.Close()
	for {
		//Loop infinito para receber mensagem e escrever todo
		//conteúdo (processo que enviou, relógio recebido e texto)
		//na tela
		//FALTA FAZER
		buf := make([]byte, 1024)
		n, _, err := Connection.ReadFromUDP(buf)
		fmt.Println("tamanho ", n)
		fmt.Println("buff ", buf[:(n+1)])
		err = json.Unmarshal(buf[:n], &processInfoReceived)
		CheckError(err)
		fmt.Println("Processo ", processInfoReceived.Id, " Pediu CS")
		time.Sleep(time.Second * 2)
		fmt.Println("Termina uso pelo Processo ", processInfoReceived.Id)
		fmt.Println("Address to send ", processInfoReceived.Address)
		processInfoReceived.TypeMessage = CSREPLY
		CliAddr, err := net.ResolveUDPAddr("udp", processInfoReceived.Address)
		CheckError(err)
		Conn, err := net.DialUDP("udp", LocalAddr, CliAddr)
		CheckError(err)
		buf, err = json.Marshal(processInfoReceived)
		CheckError(err)
		_, err = Conn.Write(buf)
		buf = nil
		if err != nil {
			fmt.Println(string(buf), err)
		}
		Conn.Close()
	}
}
