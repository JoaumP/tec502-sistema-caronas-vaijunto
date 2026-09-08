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
		fmt.Println("\n1. Publicar carona")
		fmt.Println("2. Sair")
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

func publicarCarona(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) {
	fmt.Print("Data (2006-01-02): ")
	data, _ := entrada.ReadString('\n')
	data = strings.TrimSpace(data)

	fmt.Print("Horário de saída (2006-01-02T15:04): ")
	horarioSaida, _ := entrada.ReadString('\n')
	horarioSaida = strings.TrimSpace(horarioSaida)

	var trechos []protocolo.TrechoPedido
	for {
		fmt.Print("Origem (ou vazio para terminar): ")
		origem, _ := entrada.ReadString('\n')
		origem = strings.TrimSpace(origem)
		if origem == "" {
			break
		}

		fmt.Print("Destino: ")
		destino, _ := entrada.ReadString('\n')
		destino = strings.TrimSpace(destino)

		fmt.Print("Horário de saída do trecho: ")
		hSaida, _ := entrada.ReadString('\n')
		hSaida = strings.TrimSpace(hSaida)

		fmt.Print("Horário de chegada do trecho: ")
		hChegada, _ := entrada.ReadString('\n')
		hChegada = strings.TrimSpace(hChegada)

		fmt.Print("Preço (centavos): ")
		precoStr, _ := entrada.ReadString('\n')
		preco, _ := strconv.Atoi(strings.TrimSpace(precoStr))

		fmt.Print("Assentos totais: ")
		assentosStr, _ := entrada.ReadString('\n')
		assentos, _ := strconv.Atoi(strings.TrimSpace(assentosStr))

		trechos = append(trechos, protocolo.TrechoPedido{
			Origem:         origem,
			Destino:        destino,
			HorarioSaida:   hSaida,
			HorarioChegada: hChegada,
			PrecoCentavos:  preco,
			AssentosTotais: assentos,
		})
	}

	pedido := protocolo.PedidoPublicarCarona{
		Data:         data,
		HorarioSaida: horarioSaida,
		Trechos:      trechos,
	}
	payload, _ := json.Marshal(pedido)
	msg := protocolo.Mensagem{Operacao: protocolo.OpPublicarCarona, Payload: payload}

	resposta := enviarEReceber(leitor, escritor, msg)

	if !resposta.Sucesso {
		fmt.Println("erro:", resposta.Erro)
		return
	}

	var r protocolo.RespostaPublicarCarona
	json.Unmarshal(resposta.Dados, &r)
	fmt.Println("carona publicada, ID:", r.IDCarona)
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