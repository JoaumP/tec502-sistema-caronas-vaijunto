package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"

	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/protocolo"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/servidor"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/tcp"
)

func main() {
	estado := servidor.NovoEstado()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("erro ao abrir porta:", err)
	}
	defer listener.Close()

	log.Println("servidor escutando na porta 8080")

	for {
		conexao, err := listener.Accept()
		if err != nil {
			log.Println("erro ao aceitar conexão:", err)
			continue
		}
		go atenderConexao(conexao, estado)
	}
}

func atenderConexao(conexao net.Conn, estado *servidor.Estado) {
	defer conexao.Close()

	leitor := bufio.NewReader(conexao)
	escritor := bufio.NewWriter(conexao)

	var idUsuarioLogado int64 // 0 = ninguém logado ainda, nessa conexão

	for {
		dados, err := tcp.LerMensagem(leitor)
		if err != nil {
			log.Println("cliente desconectou:", err)
			return
		}

		var msg protocolo.Mensagem
		if err := json.Unmarshal(dados, &msg); err != nil {
			enviarResposta(escritor, protocolo.Resposta{Sucesso: false, Erro: "mensagem malformada"})
			continue
		}

		var resposta protocolo.Resposta
		switch msg.Operacao {
		case protocolo.OpLogin:
			resposta = estado.HandleLogin(msg.Payload)
			if resposta.Sucesso {
				var r protocolo.RespostaLogin
				json.Unmarshal(resposta.Dados, &r)
				idUsuarioLogado = r.IDUsuario
			}
		case protocolo.OpPublicarCarona:
			resposta = estado.HandlePublicarCarona(idUsuarioLogado, msg.Payload)
		case protocolo.OpBuscarItinerario:
			resposta = estado.HandleBuscarItinerario(msg.Payload)
		case protocolo.OpConfirmarReserva:
			resposta = estado.HandleConfirmarReserva(idUsuarioLogado, msg.Payload)
		case protocolo.OpCancelarCarona:
			resposta = estado.HandleCancelarCarona(idUsuarioLogado, msg.Payload)
		case protocolo.OpConsultarCaronas:
			resposta = estado.HandleConsultarCaronas(idUsuarioLogado)
		case protocolo.OpCancelarReserva:
			resposta = estado.HandleCancelarReserva(idUsuarioLogado, msg.Payload)
		case protocolo.OpConsultarReservas:
			resposta = estado.HandleConsultarReservas(idUsuarioLogado)
		default:
			resposta = protocolo.Resposta{Sucesso: false, Erro: "operação desconhecida"}
		}

		enviarResposta(escritor, resposta)
	}
}

func enviarResposta(escritor *bufio.Writer, resposta protocolo.Resposta) {
	dados, _ := json.Marshal(resposta)
	tcp.EnviarMensagem(escritor, dados)
}