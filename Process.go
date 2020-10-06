//package ces27_lab02

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

type stateType int8

const (
	RELEASED = stateType(0)
	WANTED   = stateType(1)
	HELD     = stateType(2)
)

var stateProcess chan stateType
var numReplies chan int

type messageType int8

const (
	REQUEST   = messageType(0)
	REPLY     = messageType(1)
	CSREQUEST = messageType(2)
	CSREPLY   = messageType(3)
)

//Variáveis globais interessantes para o processo
var err string
var myPort string          //porta do meu servidor
var nServers int           //qtde de outros processo
var CliConn []*net.UDPConn //vetor com conexões para os servidores
var CSconn *net.UDPConn

//dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
//mensagens dos outros processos)

var ch chan string
var id int

type processStruct struct {
	Id          int
	Clock       int
	Address     string
	TypeMessage messageType
	Texto       string
}

var processInfo chan processStruct
var requestProcessesInfo chan []processStruct
var requestTimeStamp chan int

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
func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func incClock(clockOther int) {
	processInfoAux := <-processInfo
	processInfoAux.Clock = maxInt(processInfoAux.Clock, clockOther) + 1
	processInfo <- processInfoAux
	fmt.Println("Clock: ", processInfoAux.Clock)
}
func doServerJob() {
	var processInfoAux processStruct
	var requestProcessesInfoAux []processStruct = make([]processStruct, 0)
	var replies int
	buf := make([]byte, 1024)
	var processInfoReceived processStruct
	var message processStruct
	//var stateProcessAux stateType
	for {
		n, addr, err := ServConn.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("Error: ", err)
		}
		// processInfoReceived, erro := strconv.Atoi(string(buf[0:n]))
		err = json.Unmarshal(buf[:n], &processInfoReceived)
		CheckError(err)
		if processInfoReceived.Id != 0 {
			fmt.Printf("Received Clock = %d from Process %d and ip %s\n", processInfoReceived.Clock,
				processInfoReceived.Id, addr)
		}

		//Aumenta 1 de clock para cada mensagem recebida
		//processInfoAux.Clock = maxInt(processInfoAux.Clock, processInfoReceived.Clock) + 1
		incClock(processInfoReceived.Clock)
		processInfoAux = <-processInfo
		processInfo <- processInfoAux
		//fmt.Println("Clock ", processInfoAux.Clock, " do processo ", processInfoAux.Id)
		T := <-requestTimeStamp
		requestTimeStamp <- T
		message = processStruct{Id: processInfoAux.Id, Clock: processInfoAux.Clock, Texto: processInfoAux.Texto,
			Address: processInfoAux.Address}
		//stateProcessAux = <- stateProcess
		switch processInfoReceived.TypeMessage {
		case REQUEST:
			fmt.Println("Mensagem eh REQUEST")
			switch <-stateProcess {
			case WANTED:
				//caso seja verdade nao entra em case HELD
				fmt.Println("Process esta em WANTED")
				if T < processInfoReceived.Clock ||
					(T == processInfoReceived.Clock && processInfoAux.Id < processInfoReceived.Id) {
					message.TypeMessage = REPLY
					message.Clock = T
					fmt.Println("T = ", T)
					fmt.Println("Recebido (Tj, id) = (", processInfoReceived.Clock, ", ",
						processInfoReceived.Id, ")")
					go doClientJob(CliConn[processInfoReceived.Id-1], message)
				} else {
					fmt.Println("Recebido (Tj, id) = (", processInfoReceived.Clock, ", ",
						processInfoReceived.Id, ")")
					fmt.Println("É armazenado")
					requestProcessesInfoAux = append(requestProcessesInfoAux, <-requestProcessesInfo...)
					requestProcessesInfoAux = append(requestProcessesInfoAux, processInfoReceived)
					requestProcessesInfo <- requestProcessesInfoAux
					requestProcessesInfoAux = nil
				}
				stateProcess <- WANTED
				break
			case HELD:
				fmt.Println("Process esta em HELD")
				fmt.Println("Recebido (Tj, id) = (", processInfoReceived.Clock, ", ",
					processInfoReceived.Id, ")")
				requestProcessesInfoAux = append(requestProcessesInfoAux, <-requestProcessesInfo...)
				requestProcessesInfoAux = append(requestProcessesInfoAux, processInfoReceived)
				requestProcessesInfo <- requestProcessesInfoAux
				requestProcessesInfoAux = nil
				stateProcess <- HELD
				break
			case RELEASED:
				fmt.Println("Process esta em RELEASED")
				fmt.Println("Manda reply para o processo ", processInfoReceived.Id, " de Clock ",
					processInfoReceived.Clock)
				message.TypeMessage = REPLY
				go doClientJob(CliConn[processInfoReceived.Id-1], message)
				stateProcess <- RELEASED
				break

			}
			break
		case REPLY:
			fmt.Println("Mensagem eh REPLY")
			state := <-stateProcess
			fmt.Println("state ", state)
			switch state {
			case WANTED:
				replies = <-numReplies + 1
				fmt.Println("replies = ", replies-1)
				if replies == nServers {
					message.Texto = "Oi"
					message.TypeMessage = CSREQUEST
					fmt.Println("Entrei na CS")
					go doClientJob(CSconn, message)
					replies = 1 //comecei do 1 e nao do zero, para nao precisar fazer subtracao no if's
					state = HELD
				}
				numReplies <- replies
				break

			}
			stateProcess <- state
			break
		case CSREPLY:
			fmt.Println("Mensagem eh CS retornou")
			<-stateProcess
			stateProcess <- RELEASED
			//Libera CS e devolve para os outros processos que desejam o CS
			requestProcessesInfoAux = append(requestProcessesInfoAux, <-requestProcessesInfo...)
			nRequestMessages := len(requestProcessesInfoAux)
			message.Texto = ""
			message.TypeMessage = REPLY
			message.Clock = T
			for i := 0; i < nRequestMessages; i++ {
				fmt.Println("Manda reply para o processo ", requestProcessesInfoAux[i].Id)
				go doClientJob(CliConn[requestProcessesInfoAux[i].Id-1], message)
			}
			<-requestTimeStamp
			requestTimeStamp <- processInfoAux.Clock
			requestProcessesInfoAux = nil
			requestProcessesInfo <- requestProcessesInfoAux
			break

		}

		//processInfoAux.Clock++
		fmt.Printf("Logical Clock: %d\n", processInfoAux.Clock)
		//stateProcess <- stateProcessAux
	}
	//Loop infinito
	// for {
	// 	//Ler (uma vez somente) da conexão UDP a mensagem
	// 	//Escrever na tela a msg recebida (indicando o endereço de quem enviou)
	// }
}
func doClientJob(conn *net.UDPConn, message processStruct) {
	//Enviar uma mensagem (com valor i) para o servidor do processo
	//otherServer
	//fmt.Println("Flag01")
	incClock(0)
	buf, err := json.Marshal(message)
	CheckError(err)
	fmt.Println(string(buf), err)
	_, err = conn.Write(buf)
	if err != nil {
		fmt.Println(string(buf), err)
	}
	//time.Sleep(time.Second * 1)
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
	myAddressString := "127.0.0.1" + myPort
	ServerAddr, err := net.ResolveUDPAddr("udp", myAddressString)
	CheckError(err)
	clockAux := processStruct{Id: id, Clock: 0, Address: myAddressString, Texto: ""}
	fmt.Println("Address ", clockAux.Address)
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)
	LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	CheckError(err)
	//Outros códigos para deixar ok as conexões com os servidores dos outros processos. Colocar tais conexões no vetor CliConn.
	CliConn = make([]*net.UDPConn, nServers) //Aloca vetor
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

	CliAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	CheckError(err)
	CSconn, err = net.DialUDP("udp", LocalAddr, CliAddr)
	CheckError(err)

	processInfo <- clockAux
	stateProcess <- RELEASED
	requestProcessesInfoAux := make([]processStruct, 0)
	requestProcessesInfo <- requestProcessesInfoAux
	numReplies <- 1 //comecei do 1 e nao do zero, para nao precisar fazer subtracao no if's
	requestTimeStamp <- 0
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
	var processInfoAux processStruct
	//var message processStruct
	processInfo = make(chan processStruct, 1)
	stateProcess = make(chan stateType, 1)
	requestProcessesInfo = make(chan []processStruct, 1)
	requestTimeStamp = make(chan int, 1)
	numReplies = make(chan int, 1)
	// fmt.Println("Main Flag01")
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
				if x == "x" {
					switch <-stateProcess {
					case WANTED:
						stateProcess <- WANTED
						fmt.Print("x ignorado")
						break
					case RELEASED:
						stateProcess <- WANTED
						fmt.Println("Processo foi de RELEASED para WANTED")

						processInfoAux = <-processInfo
						processInfo <- processInfoAux
						T := processInfoAux.Clock
						<-requestTimeStamp
						requestTimeStamp <- processInfoAux.Clock
						//Espera 5 segundos para começar os pedidos
						fmt.Println("Espera 5 segundos para mandar requests")
						time.Sleep(time.Second * 5)
						fmt.Println("O agent faz os requests para pedir o direito ao CS")
						fmt.Println("T =", T, " para o processo ", id)
						processInfoAux.TypeMessage = REQUEST
						for i := 1; i < id; i++ {
							//processInfoAux.TypeMessage = REQUEST
							go doClientJob(CliConn[i-1], processInfoAux)
						}
						// Eu escolhi ter dois for's do que ter if's no dentro de todos os loops
						for i := id + 1; i <= nServers; i++ {
							//processInfoAux.TypeMessage = REQUEST
							go doClientJob(CliConn[i-1], processInfoAux)
						}
						//processInfo <- processInfoAux
						break
					case HELD:
						fmt.Print("x ignorado")
						stateProcess <- HELD
						break
					}

				} else {
					idEnviar, erro := strconv.Atoi(x)
					CheckError(erro)
					//processInfoAux = <-processInfo
					//processInfoAux.Clock++
					if id == idEnviar {
						// go doClientJob(id-1, id)
						//fmt.Printf("Logical Clock: %d\n", processInfoAux.Clock)
						incClock(0)
					} else {

						fmt.Println("id nao corresponde ao do processo")
					}
					//processInfo <- processInfoAux
				}

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
