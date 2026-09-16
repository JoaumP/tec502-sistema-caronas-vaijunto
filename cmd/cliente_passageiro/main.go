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
	var ultimaBusca []protocolo.ItinerarioEncontrado

	for {
		auxiliares.ImprimirMenu("Menu Passageiro", []string{
			"Buscar itinerário",
			"Confirmar reserva",
			"Consultar minhas reservas",
			"Cancelar reserva",
			"Sair da conta",
		})
		opcao, _ := entrada.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		switch opcao {
		case "1":
			ultimaBusca = buscarItinerario(leitor, escritor, entrada)
		case "2":
			confirmarReserva(leitor, escritor, entrada, ultimaBusca)
		case "3":
			consultarReservas(leitor, escritor, entrada)
		case "4":
			cancelarReserva(leitor, escritor, entrada)
		case "5":
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

	fmt.Println()
	for i, it := range r.Itinerarios {
		auxiliares.ImprimirSeparador()
		fmt.Printf("Opção %d — R$ %.2f — %d trecho(s)\n", i+1, float64(it.PrecoCentavos)/100, len(it.Trechos))
		for _, t := range it.Trechos {
			fmt.Printf("   %s (%s)  →  %s (%s)\n", t.Origem, auxiliares.FormatarHorario(t.HorarioSaida), t.Destino, auxiliares.FormatarHorario(t.HorarioChegada))
		}
	}
	auxiliares.ImprimirSeparador()

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

	var idsTrecho []int64
	for _, t := range ultimaBusca[numero-1].Trechos {
		idsTrecho = append(idsTrecho, t.ID)
	}

	pedido := protocolo.PedidoConfirmarReserva{Itinerario: idsTrecho}
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

func consultarReservas(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	msg := protocolo.Mensagem{Operacao: protocolo.OpConsultarReservas}
	resposta := auxiliares.EnviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	var r protocolo.RespostaConsultarReservas
	json.Unmarshal(resposta.Dados, &r)

	if len(r.Reservas) == 0 {
		fmt.Println("você ainda não tem nenhuma reserva")
		return
	}

	fmt.Println()
	for _, res := range r.Reservas {
		auxiliares.ImprimirSeparador()
		fmt.Printf("Reserva #%d — status: %s\n", res.ID, strings.ToUpper(res.Status))
		for _, t := range res.Trechos {
			fmt.Printf("   %s (%s)  →  %s (%s)\n", t.Origem, auxiliares.FormatarHorario(t.HorarioSaida), t.Destino, auxiliares.FormatarHorario(t.HorarioChegada))
		}
	}
	auxiliares.ImprimirSeparador()
}

func cancelarReserva(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	id := auxiliares.LerInteiroPositivo(entrada, "ID da reserva a cancelar: ")

	pedido := protocolo.PedidoCancelarReserva{IDReserva: int64(id)}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpCancelarReserva, Payload: payload}

	resposta := auxiliares.EnviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	fmt.Println("reserva cancelada com sucesso")
}
