package auxiliares

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/protocolo"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/tcp"

)

func Autenticar(leitor *bufio.Reader, escritor *bufio.Writer, entrada *bufio.Reader) bool {
	for {
		fmt.Println("\n1. Entrar")
		fmt.Println("2. Cadastrar")
		fmt.Println("3. Sair")
		fmt.Print("> ")
		opcao, _ := entrada.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		var operacao string
		switch opcao {
		case "1":
			operacao = protocolo.OpLogin
		case "2":
			operacao = protocolo.OpCadastro
		case "3":
			return false
		default:
			fmt.Println("opção inválida")
			continue
		}

		fmt.Print("Login: ")
		login, _ := entrada.ReadString('\n')
		login = strings.TrimSpace(login)

		fmt.Print("Senha: ")
		senha, _ := entrada.ReadString('\n')
		senha = strings.TrimSpace(senha)

		pedido := protocolo.PedidoLogin{Login: login, Senha: senha}
		payload, _ := json.Marshal(pedido)
		msg := protocolo.Mensagem{Operacao: operacao, Payload: payload}

		resposta := EnviarEReceber(leitor, escritor, msg)

		if !resposta.Sucesso {
			fmt.Println("erro:", resposta.Erro)
			continue
		}

		fmt.Println("sucesso!")
		return true
	}
}

func EnviarEReceber(leitor *bufio.Reader, escritor *bufio.Writer, msg protocolo.Mensagem) protocolo.Resposta {
	dados, _ := json.Marshal(msg)
	tcp.EnviarMensagem(escritor, dados)

	respostaBytes, err := tcp.LerMensagem(leitor)
	if err != nil {
		fmt.Println("erro ao ler resposta:", err)
		return protocolo.Resposta{Sucesso: false, Erro: "falha de conexão"}
	}

	var resposta protocolo.Resposta
	json.Unmarshal(respostaBytes, &resposta)
	return resposta
}