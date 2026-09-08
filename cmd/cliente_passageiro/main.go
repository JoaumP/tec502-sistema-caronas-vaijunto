package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/protocolo"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/tcp"
)

func main() {
	conexao, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("erro ao conectar:", err)
		return
	}
	defer conexao.Close()

	leitor := bufio.NewReader(conexao)
	escritor := bufio.NewWriter(conexao)
	entrada := bufio.NewReader(os.Stdin)

	fazerLogin(leitor, escritor, entrada)

	for {
		fmt.Println("\n1. Buscar itinerário")
		fmt.Println("2. Confirmar reserva")
		fmt.Println("3. Sair")
		fmt.Print("> ")
		opcao, _ := entrada.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		switch opcao {
		case "1":
			buscarItinerario(leitor, escritor, entrada)
		case "2":
			confirmarReserva(leitor, escritor, entrada)
		case "3":
			return
		default:
			fmt.Println("opção inválida")
		}
	}
}

func fazerLogin(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	fmt.Print("Login: ")
	login, _ := entrada.ReadString('\n')
	login = strings.TrimSpace(login)

	fmt.Print("Senha: ")
	senha, _ := entrada.ReadString('\n')
	senha = strings.TrimSpace(senha)

	pedido := protocolo.PedidoLogin{Login: login, Senha: senha}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpLogin, Payload: payload}

	resposta := enviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro no login:", resposta.Erro)
		os.Exit(1)
	}
	fmt.Println("login realizado com sucesso")
}

func buscarItinerario(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	fmt.Print("Origem: ")
	origem, _ := entrada.ReadString('\n')
	origem = strings.TrimSpace(origem)

	fmt.Print("Destino: ")
	destino, _ := entrada.ReadString('\n')
	destino = strings.TrimSpace(destino)

	fmt.Print("Data (2006-01-02): ")
	data, _ := entrada.ReadString('\n')
	data = strings.TrimSpace(data)

	pedido := protocolo.PedidoBuscarItinerario{Origem: origem, Destino: destino, Data: data}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpBuscarItinerario, Payload: payload}

	resposta := enviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	var r protocolo.RespostaBuscarItinerario
	json.Unmarshal(resposta.Dados, &r)

	if len(r.Itinerarios) == 0 {
		fmt.Println("nenhum itinerário encontrado")
		return
	}

	for i, it := range r.Itinerarios {
		fmt.Printf("%d) trechos: %v | preço: %d centavos\n", i+1, it.Trechos, it.PrecoCentavos)
	}
}

func confirmarReserva(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	fmt.Print("IDs dos trechos, separados por vírgula: ")
	linha, _ := entrada.ReadString('\n')
	linha = strings.TrimSpace(linha)

	partes := strings.Split(linha, ",")
	var itinerario []int64
	for _, p := range partes {
		id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			fmt.Println("ID inválido:", p)
			return
		}
		itinerario = append(itinerario, id)
	}

	pedido := protocolo.PedidoConfirmarReserva{Itinerario: itinerario}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpConfirmarReserva, Payload: payload}

	resposta := enviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	var r protocolo.RespostaConfirmarReserva
	json.Unmarshal(resposta.Dados, &r)
	fmt.Println("reserva confirmada, ID:", r.IDReserva)
}

func enviarEReceber(leitor *bufio.Reader, escritor *bufio.Writer, msg protocolo.Mensagem) protocolo.Resposta {
	dados, _ := json.Marshal(msg)
	tcp.EnviarMensagem(escritor, dados)

	respostaBytes, err := tcp.LerMensagem(leitor)
	if err != nil {
		fmt.Println("erro ao ler resposta:", err)
		os.Exit(1)
	}

	var resposta protocolo.Resposta
	json.Unmarshal(respostaBytes, &resposta)
	return resposta
}