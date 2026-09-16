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
	for {
		auxiliares.ImprimirMenu("Menu Motorista", []string{
			"Publicar carona",
			"Consultar minhas caronas",
			"Cancelar carona",
			"Sair da conta",
		})
		opcao, _ := entrada.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		switch opcao {
		case "1":
			publicarCarona(leitor, escritor, entrada)
		case "2":
			consultarCaronas(leitor, escritor, entrada)
		case "3":
			cancelarCarona(leitor, escritor, entrada)
		case "4":
			return
		default:
			fmt.Println("opção inválida")
		}
	}
}

func publicarCarona(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	auxiliares.ImprimirTitulo("Publicar Carona")
	fmt.Println("Cadastre a rota, trecho por trecho.")

	var trechos []protocolo.TrechoPedido
	origemAtual := auxiliares.LerTexto(entrada, "Cidade de origem (partida da rota): ")

	var dataAtual string
	numero := 1

	for {
		destino := auxiliares.LerTexto(entrada, "Cidade de destino deste trecho: ")

		if numero == 1 {
			dataTexto := auxiliares.LerData(entrada, "Data (DD/MM/AAAA): ")
			dataAtual, _ = auxiliares.ConverterData(dataTexto)
		}

		horaSaida := auxiliares.LerHora(entrada, "Horário de saída (HH:MM): ")
		horaChegada := auxiliares.LerHora(entrada, "Horário de chegada (HH:MM): ")

		hSaida, hChegada, err := auxiliares.MontarHorarios(dataAtual, horaSaida, horaChegada)
		if err != nil {
			fmt.Println("erro ao montar horário:", err)
			return
		}

		preco := auxiliares.LerPrecoCentavos(entrada, "Preço (R$): ")
		assentos := auxiliares.LerInteiroPositivo(entrada, "Assentos totais: ")

		trechos = append(trechos, protocolo.TrechoPedido{
			Origem:         origemAtual,
			Destino:        destino,
			HorarioSaida:   hSaida,
			HorarioChegada: hChegada,
			PrecoCentavos:  preco,
			AssentosTotais: assentos,
		})

		if !auxiliares.LerSimNao(entrada, "\nAdicionar outro trecho a partir de "+destino+"? (s/n): ") {
			break
		}

		origemAtual = destino
		dataAtual = hChegada[:10]
		numero++
	}

	pedido := protocolo.PedidoPublicarCarona{Trechos: trechos}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpPublicarCarona, Payload: payload}

	resposta := auxiliares.EnviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	var r protocolo.RespostaPublicarCarona
	json.Unmarshal(resposta.Dados, &r)
	fmt.Printf("\n✔ Carona publicada com sucesso! ID: %d\n", r.IDCarona)
}

func consultarCaronas(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	msg := protocolo.Mensagem{Operacao: protocolo.OpConsultarCaronas}
	resposta := auxiliares.EnviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	var r protocolo.RespostaConsultarCaronas
	json.Unmarshal(resposta.Dados, &r)

	if len(r.Caronas) == 0 {
		fmt.Println("você ainda não publicou nenhuma carona")
		return
	}

	fmt.Println()
	for _, c := range r.Caronas {
		auxiliares.ImprimirSeparador()
		fmt.Printf("Carona #%d — status: %s\n", c.ID, strings.ToUpper(c.Status))
		for _, t := range c.Trechos {
			fmt.Printf("   %s (%s)  →  %s (%s)  |  vagas: %d/%d  |  passageiros: %v\n",
				t.Origem, auxiliares.FormatarHorario(t.HorarioSaida), t.Destino, auxiliares.FormatarHorario(t.HorarioChegada),
				t.AssentosLivres, t.AssentosTotais, t.Passageiros)
		}
	}
	auxiliares.ImprimirSeparador()
}

func cancelarCarona(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	id := auxiliares.LerInteiroPositivo(entrada, "ID da carona a cancelar: ")

	pedido := protocolo.PedidoCancelarCarona{IDCarona: int64(id)}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpCancelarCarona, Payload: payload}

	resposta := auxiliares.EnviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	fmt.Println("carona cancelada com sucesso")
}