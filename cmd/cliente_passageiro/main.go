package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/protocolo"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/auxiliares"
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

	for {
		if !auxiliares.Autenticar(leitor, escritor, entrada) {
			return
		}
		menuPrincipal(leitor, escritor, entrada)
	}
}

func menuPrincipal(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	var ultimaBusca []protocolo.ItinerarioEncontrado // guarda a última busca feita

	for {
		fmt.Println("\n1. Buscar itinerário")
		fmt.Println("2. Confirmar reserva")
		fmt.Println("3. Sair da conta")
		fmt.Print("> ")
		opcao, _ := entrada.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		switch opcao {
		case "1":
			ultimaBusca = buscarItinerario(leitor, escritor, entrada)
		case "2":
			confirmarReserva(leitor, escritor, entrada, ultimaBusca)
		case "3":
			return
		default:
			fmt.Println("opção inválida")
		}
	}
}

func buscarItinerario(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) []protocolo.ItinerarioEncontrado {
	origem := auxiliares.LerTexto(entrada, "Origem: ")
	destino := auxiliares.LerTexto(entrada, "Destino: ")
	dataTexto := auxiliares.LerData(entrada, "Data (DD/MM/AAAA): ")
	data, _ := auxiliares.ConverterData(dataTexto)

	pedido := protocolo.PedidoBuscarItinerario{Origem: origem, Destino: destino, Data: data}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpBuscarItinerario, Payload: payload}

	resposta := auxiliares.EnviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return nil
	}

	var r protocolo.RespostaBuscarItinerario
	json.Unmarshal(resposta.Dados, &r)

	if len(r.Itinerarios) == 0 {
		fmt.Println("nenhum itinerário encontrado")
		return nil
	}

	for i, it := range r.Itinerarios {
		fmt.Printf("%d) trechos: %v | preço: R$ %.2f\n", i+1, it.Trechos, float64(it.PrecoCentavos)/100)
	}

	return r.Itinerarios
}

func confirmarReserva(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader, ultimaBusca []protocolo.ItinerarioEncontrado) {
	if len(ultimaBusca) == 0 {
		fmt.Println("faça uma busca primeiro")
		return
	}

	numero := auxiliares.LerInteiroPositivo(entrada, "Número da opção desejada: ")
	if numero < 1 || numero > len(ultimaBusca) {
		fmt.Println("opção inválida")
		return
	}

	itinerario := ultimaBusca[numero-1].Trechos

	pedido := protocolo.PedidoConfirmarReserva{Itinerario: itinerario}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpConfirmarReserva, Payload: payload}

	resposta := auxiliares.EnviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	var r protocolo.RespostaConfirmarReserva
	json.Unmarshal(resposta.Dados, &r)
	fmt.Println("reserva confirmada, ID:", r.IDReserva)
}