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
		fmt.Println("\n1. Publicar carona")
		fmt.Println("2. Sair da conta")
		fmt.Print("> ")
		opcao, _ := entrada.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		switch opcao {
		case "1":
			publicarCarona(leitor, escritor, entrada)
		case "2":
			return
		default:
			fmt.Println("opção inválida")
		}
	}
}

func publicarCarona(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	fmt.Println("Cadastre a rota, trecho por trecho.")

	var trechos []protocolo.TrechoPedido
	origemAtual := auxiliares.LerTexto(entrada, "Cidade de origem (partida da rota): ")

	for {
		destino := auxiliares.LerTexto(entrada, "Cidade de destino deste trecho: ")
		data := auxiliares.LerData(entrada, "Data (DD/MM/AAAA): ")
		horaSaida := auxiliares.LerHora(entrada, "Horário de saída (HH:MM): ")
		horaChegada := auxiliares.LerHora(entrada, "Horário de chegada (HH:MM): ")

		hSaida, hChegada, err := auxiliares.MontarHorarios(data, horaSaida, horaChegada)
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
	fmt.Println("carona publicada, ID:", r.IDCarona)
}