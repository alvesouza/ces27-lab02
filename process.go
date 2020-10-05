//package ces27_lab02
package main
import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

//Variáveis globais interessantes para o processo
var err string
var myPort string          //porta do meu servidor
var nServers int           //qtde de outros processo
var CliConn []*net.UDPConn //vetor com conexões para os servidores
//dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
//mensagens dos outros processos)

var ch chan string
var logicalClock chan int
var id int

func CheckError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}
func PrintError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
	}
}
func doServerJob() {
	buf := make([]byte, 1024)

	for {
		n, addr, err := ServConn.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("Error: ", err)
		}
		clockRecebido, erro := strconv.Atoi(string(buf[0:n]))
		CheckError(erro)

		fmt.Println("Received ", clockRecebido, " from ", addr)

		clockAux := <-logicalClock
		if clockAux < clockRecebido {
			clockAux = clockRecebido
		}
		clockAux = clockAux + 1
		fmt.Println("Logical Clock: ", clockAux)
		logicalClock <- clockAux
	}
	//Loop infinito
	// for {
	// 	//Ler (uma vez somente) da conexão UDP a mensagem
	// 	//Escrever na tela a msg recebida (indicando o endereço de quem enviou)
	// }
}
func doClientJob(otherProcess int, i int) {
	//Enviar uma mensagem (com valor i) para o servidor do processo
	//otherServer
	msg := strconv.Itoa(i)
	fmt.Println("Mensagem enviada para o id ", otherProcess+1, ": ", msg)
	buf := []byte(msg)
	_, err := CliConn[otherProcess].Write(buf)
	if err != nil {
		fmt.Println(msg, err)
	}
	time.Sleep(time.Second * 1)
}

func initConnections() {
	/*Esse 2 tira o nome (no caso Process) e tira a primeira porta
	(que é a minha). As demais portas são dos outros processos*/
	id, _ = strconv.Atoi(os.Args[1])
	// CheckError(err)
	myPort = os.Args[id+1]
	nServers = len(os.Args) - 2
	// fmt.Println("Flag01")
	// fmt.Println("Server: ", "127.0.0.1"+myPort)
	//Outros códigos para deixar ok a conexão do meu servidor (onde recebo msgs). O processo já deve ficar habilitado a receber msgs.
	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
	CheckError(err)

	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)
	LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	CheckError(err)
	//Outros códigos para deixar ok as conexões com os servidores dos outros processos. Colocar tais conexões no vetor CliConn.

	CliConn = make([]*net.UDPConn, nServers) //Aluca vetor
	for i := 0; i < nServers; i++ {
		// if (id + 1) != i+2 {
		// fmt.Println("Server para enviar: ", "127.0.0.1"+os.Args[i+2])
		CliAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+os.Args[i+2])
		CheckError(err)
		Conn, err := net.DialUDP("udp", LocalAddr, CliAddr)
		CheckError(err)
		CliConn[i] = Conn
		// }

	}
	// nServers--
}

func readInput(ch chan string) {
	// Non-blocking async routine to listen for terminal input
	// fmt.Println("readInput Flag01")
	reader := bufio.NewReader(os.Stdin)
	// fmt.Println("readInput Flag02")
	for {
		// fmt.Println("readInput Flag03")
		text, _, _ := reader.ReadLine()
		ch <- string(text)
		// fmt.Println("readInput Flag04")
	}
}

func main() {
	logicalClock = make(chan int, 1)
	// fmt.Println("Main Flag01")
	logicalClock <- 0
	// fmt.Println("Main Flag01.01")
	initConnections()
	//O fechamento de conexões deve ficar aqui, assim só fecha
	//conexão quando a main morrer
	ch = make(chan string)
	defer ServConn.Close()
	// fmt.Println("Main Flag02")
	for i := 0; i < nServers; i++ {
		// fmt.Println("Main Flag03")
		defer CliConn[i].Close()
	}
	//Todo Process fará a mesma coisa: ficar ouvindo mensagens e mandar infinitos i’s para os outros processos
	go readInput(ch)
	// fmt.Println("Main Flag04")
	go doServerJob()
	// fmt.Println("Main Flag05")
	for {
		// When there is a request (from stdin). Do it!
		select {
		case x, valid := <-ch:
			if valid {
				fmt.Printf("Recebi do teclado: %s \n", x)
				idEnviar, erro := strconv.Atoi(x)
				CheckError(erro)
				logicalClockAux := (<-logicalClock) + 1
				if id == idEnviar {
					// go doClientJob(id-1, id)
					// logicalClock <- (<-logicalClock) + 1
					fmt.Println("Logical Clock: ", logicalClockAux)
				} else if idEnviar > 0 && idEnviar <= nServers {
					// logicalClock <- (<-logicalClock) + 1
					go doClientJob(idEnviar-1, logicalClockAux)
				} else {
					fmt.Println("id invalido")
				}
				logicalClock <- logicalClockAux
			} else {
				fmt.Println("Channel closed!")
			}
		default:
			// Do nothing in the non-blocking approach.
			time.Sleep(time.Second * 1)
		}
		// Wait a while
		time.Sleep(time.Second * 1)

	}
}