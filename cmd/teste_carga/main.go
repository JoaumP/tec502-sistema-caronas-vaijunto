package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/protocolo"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/tcp"
)

var sufixo = fmt.Sprintf("%d", time.Now().UnixNano())

func enderecoServidor() string {
	if e := os.Getenv("SERVIDOR"); e != "" {
		return e
	}
	return "localhost:8080"
}

func conectar() (*bufio.Reader, *bufio.Writer, net.Conn) {
	conexao, err := net.Dial("tcp", enderecoServidor())
	if err != nil {
		fmt.Println("erro ao conectar:", err)
		os.Exit(1)
	}
	return bufio.NewReader(conexao), bufio.NewWriter(conexao), conexao
}

func enviarEReceber(leitor *bufio.Reader, escritor *bufio.Writer, msg protocolo.Mensagem) protocolo.Resposta {
	dados, _ := json.Marshal(msg)
	tcp.EnviarMensagem(escritor, dados)

	respostaBytes, err := tcp.LerMensagem(leitor)
	if err != nil {
		return protocolo.Resposta{Sucesso: false, Erro: "falha de conexão: " + err.Error()}
	}

	var resposta protocolo.Resposta
	json.Unmarshal(respostaBytes, &resposta)
	return resposta
}

func novoCliente(nomeBase string) (*bufio.Reader, *bufio.Writer, net.Conn) {
	leitor, escritor, conexao := conectar()
	login := nomeBase + "_" + sufixo
	payload, _ := json.Marshal(protocolo.PedidoLogin{Login: login, Senha: "123"})
	r := enviarEReceber(leitor, escritor, protocolo.Mensagem{Operacao: protocolo.OpCadastro, Payload: payload})
	if !r.Sucesso {
		fmt.Println("erro ao cadastrar", login, ":", r.Erro)
		os.Exit(1)
	}
	return leitor, escritor, conexao
}

func publicarCarona(leitor *bufio.Reader, escritor *bufio.Writer, trechos []protocolo.TrechoPedido) (int64, []protocolo.TrechoDetalhado) {
	pedido := protocolo.PedidoPublicarCarona{Trechos: trechos}
	payload, _ := json.Marshal(pedido)
	resp := enviarEReceber(leitor, escritor, protocolo.Mensagem{Operacao: protocolo.OpPublicarCarona, Payload: payload})
	if !resp.Sucesso {
		fmt.Println("erro ao publicar carona:", resp.Erro)
		os.Exit(1)
	}
	var r protocolo.RespostaPublicarCarona
	json.Unmarshal(resp.Dados, &r)

	respConsulta := enviarEReceber(leitor, escritor, protocolo.Mensagem{Operacao: protocolo.OpConsultarCaronas})
	var consulta protocolo.RespostaConsultarCaronas
	json.Unmarshal(respConsulta.Dados, &consulta)

	for _, c := range consulta.Caronas {
		if c.ID == r.IDCarona {
			return c.ID, c.Trechos
		}
	}
	fmt.Println("carona publicada não encontrada na consulta")
	os.Exit(1)
	return 0, nil
}

func confirmarReserva(leitor *bufio.Reader, escritor *bufio.Writer, itinerario []int64) protocolo.Resposta {
	pedido := protocolo.PedidoConfirmarReserva{Itinerario: itinerario}
	payload, _ := json.Marshal(pedido)
	return enviarEReceber(leitor, escritor, protocolo.Mensagem{Operacao: protocolo.OpConfirmarReserva, Payload: payload})
}

var totalTestes, testesOk int

func checar(nome string, condicao bool, detalhe string) {
	totalTestes++
	if condicao {
		testesOk++
		fmt.Printf("✔ %s\n", nome)
	} else {
		fmt.Printf("✘ %s — %s\n", nome, detalhe)
	}
}

func main() {
	fmt.Println("=== Teste de carga e corretude — VAIJUNTO ===")
	fmt.Println("execução:", sufixo)

	testeVendaDupla()
	testeAtomicidadeTrechoInexistente()
	testeItinerarioParcialSobConcorrencia()
	testeIDsDuplicadosNoItinerario()
	testeCancelamentoLiberaVagasMultiMotorista()
	testeMensagemMalformada()
	testeDeadlockEntreMotoristasDiferentes()

	fmt.Printf("\n=== Resumo: %d/%d testes passaram ===\n", testesOk, totalTestes)
}

// 1. Múltiplos clientes disputando o mesmo trecho: nenhum assento vendido duas vezes
func testeVendaDupla() {
	fmt.Println("\n--- Teste 1: venda dupla sob concorrência ---")

	const assentos = 5
	const passageiros = 20

	leitorM, escritorM, conexaoM := novoCliente("motorista_venda_dupla")
	defer conexaoM.Close()

	_, trechos := publicarCarona(leitorM, escritorM, []protocolo.TrechoPedido{
		{Origem: "A1", Destino: "B1", HorarioSaida: "2026-12-01T08:00", HorarioChegada: "2026-12-01T10:00", PrecoCentavos: 5000, AssentosTotais: assentos},
	})
	idTrecho := trechos[0].ID

	var wg sync.WaitGroup
	var sucessos int64
	tempos := make([]time.Duration, passageiros)

	for i := 0; i < passageiros; i++ {
		wg.Add(1)
		go func(indice int) {
			defer wg.Done()
			leitor, escritor, conexao := novoCliente(fmt.Sprintf("passageiro_vd_%d", indice))
			defer conexao.Close()

			inicio := time.Now()
			resp := confirmarReserva(leitor, escritor, []int64{idTrecho})
			tempos[indice] = time.Since(inicio)

			if resp.Sucesso {
				atomic.AddInt64(&sucessos, 1)
			}
		}(i)
	}
	wg.Wait()

	checar("número de sucessos bate com assentos disponíveis",
		sucessos == assentos,
		fmt.Sprintf("esperado %d, obtido %d", assentos, sucessos))

	sort.Slice(tempos, func(i, j int) bool { return tempos[i] < tempos[j] })
	var soma time.Duration
	for _, t := range tempos {
		soma += t
	}
	fmt.Printf("  tempo de resposta — mín: %v | médio: %v | máx: %v\n",
		tempos[0], soma/time.Duration(passageiros), tempos[len(tempos)-1])
}

// 2. Itinerário com trecho inexistente deve ser rejeitado por inteiro
func testeAtomicidadeTrechoInexistente() {
	fmt.Println("\n--- Teste 2: itinerário com trecho inexistente ---")

	leitorM, escritorM, conexaoM := novoCliente("motorista_atomicidade1")
	defer conexaoM.Close()

	_, trechos := publicarCarona(leitorM, escritorM, []protocolo.TrechoPedido{
		{Origem: "A2", Destino: "B2", HorarioSaida: "2026-12-01T08:00", HorarioChegada: "2026-12-01T09:00", PrecoCentavos: 1000, AssentosTotais: 3},
	})
	idValido := trechos[0].ID
	idInexistente := int64(99999999)

	leitorP, escritorP, conexaoP := novoCliente("passageiro_atomicidade1")
	defer conexaoP.Close()

	resp := confirmarReserva(leitorP, escritorP, []int64{idValido, idInexistente})
	checar("reserva com trecho inexistente é rejeitada", !resp.Sucesso, "deveria ter falhado")

	respVerifica := confirmarReserva(leitorP, escritorP, []int64{idValido})
	checar("trecho válido continua disponível após rejeição", respVerifica.Sucesso, respVerifica.Erro)
}

// 3. Itinerário multi-trecho perdendo a disputa por um trecho: o outro trecho deve ser liberado
func testeItinerarioParcialSobConcorrencia() {
	fmt.Println("\n--- Teste 3: itinerário parcial sob concorrência real ---")

	leitorM, escritorM, conexaoM := novoCliente("motorista_parcial")
	defer conexaoM.Close()

	_, trechos := publicarCarona(leitorM, escritorM, []protocolo.TrechoPedido{
		{Origem: "X3", Destino: "Y3", HorarioSaida: "2026-12-02T08:00", HorarioChegada: "2026-12-02T09:00", PrecoCentavos: 1000, AssentosTotais: 1},
		{Origem: "Y3", Destino: "Z3", HorarioSaida: "2026-12-02T09:00", HorarioChegada: "2026-12-02T10:00", PrecoCentavos: 1000, AssentosTotais: 1},
	})
	idTrecho1, idTrecho2 := trechos[0].ID, trechos[1].ID

	var wg sync.WaitGroup
	resultados := make([]protocolo.Resposta, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		leitor, escritor, conexao := novoCliente("passageiro_rouba_vaga")
		defer conexao.Close()
		resultados[0] = confirmarReserva(leitor, escritor, []int64{idTrecho2})
	}()
	go func() {
		defer wg.Done()
		leitor, escritor, conexao := novoCliente("passageiro_completo")
		defer conexao.Close()
		resultados[1] = confirmarReserva(leitor, escritor, []int64{idTrecho1, idTrecho2})
	}()
	wg.Wait()

	ambosSucesso := resultados[0].Sucesso && resultados[1].Sucesso
	checar("nunca os dois disputantes do trecho 2 têm sucesso ao mesmo tempo", !ambosSucesso, "vendeu o mesmo assento duas vezes")

	if !resultados[1].Sucesso {
		leitor, escritor, conexao := novoCliente("passageiro_verifica_liberacao")
		defer conexao.Close()
		resp := confirmarReserva(leitor, escritor, []int64{idTrecho1})
		checar("trecho 1 foi liberado após itinerário completo falhar", resp.Sucesso, resp.Erro)
	} else {
		fmt.Println("  (itinerário completo venceu a corrida desta vez — resultado válido também)")
	}
}

// 4. Itinerário com o mesmo trecho repetido deve ser rejeitado
func testeIDsDuplicadosNoItinerario() {
	fmt.Println("\n--- Teste 4: IDs de trecho duplicados no itinerário ---")

	leitorM, escritorM, conexaoM := novoCliente("motorista_duplicado")
	defer conexaoM.Close()

	_, trechos := publicarCarona(leitorM, escritorM, []protocolo.TrechoPedido{
		{Origem: "A4", Destino: "B4", HorarioSaida: "2026-12-01T08:00", HorarioChegada: "2026-12-01T09:00", PrecoCentavos: 1000, AssentosTotais: 3},
	})
	id := trechos[0].ID

	leitorP, escritorP, conexaoP := novoCliente("passageiro_duplicado")
	defer conexaoP.Close()

	resp := confirmarReserva(leitorP, escritorP, []int64{id, id})
	checar("itinerário com trecho repetido é rejeitado", !resp.Sucesso, "deveria ter falhado")
}

// 5. Cancelar carona de um motorista deve liberar vagas mesmo em trechos de outro motorista
// dentro do mesmo itinerário do passageiro
func testeCancelamentoLiberaVagasMultiMotorista() {
	fmt.Println("\n--- Teste 5: cancelamento de carona libera itinerário multi-motorista ---")

	leitorM1, escritorM1, conexaoM1 := novoCliente("motorista_A5")
	defer conexaoM1.Close()
	idCaronaA, trechosA := publicarCarona(leitorM1, escritorM1, []protocolo.TrechoPedido{
		{Origem: "X5", Destino: "Y5", HorarioSaida: "2026-12-03T08:00", HorarioChegada: "2026-12-03T09:00", PrecoCentavos: 1000, AssentosTotais: 1},
	})

	leitorM2, escritorM2, conexaoM2 := novoCliente("motorista_B5")
	defer conexaoM2.Close()
	_, trechosB := publicarCarona(leitorM2, escritorM2, []protocolo.TrechoPedido{
		{Origem: "Y5", Destino: "Z5", HorarioSaida: "2026-12-03T09:00", HorarioChegada: "2026-12-03T10:00", PrecoCentavos: 1000, AssentosTotais: 1},
	})

	idTrechoA, idTrechoB := trechosA[0].ID, trechosB[0].ID

	leitorP, escritorP, conexaoP := novoCliente("passageiro_multimotorista")
	defer conexaoP.Close()
	resp := confirmarReserva(leitorP, escritorP, []int64{idTrechoA, idTrechoB})
	checar("itinerário multi-motorista confirmado", resp.Sucesso, resp.Erro)

	// motorista A cancela a carona dele
	pedidoCancelar := protocolo.PedidoCancelarCarona{IDCarona: idCaronaA}
	payload, _ := json.Marshal(pedidoCancelar)
	respCancelar := enviarEReceber(leitorM1, escritorM1, protocolo.Mensagem{Operacao: protocolo.OpCancelarCarona, Payload: payload})
	checar("cancelamento da carona A é aceito", respCancelar.Sucesso, respCancelar.Erro)

	// trecho B (de outro motorista) deveria ter sido liberado também
	leitorP2, escritorP2, conexaoP2 := novoCliente("passageiro_verifica_liberacao_B")
	defer conexaoP2.Close()
	respNovaReserva := confirmarReserva(leitorP2, escritorP2, []int64{idTrechoB})
	checar("trecho do motorista B foi liberado ao cancelar carona do motorista A", respNovaReserva.Sucesso, respNovaReserva.Erro)
}

// 6. Mensagem malformada não deve derrubar o servidor nem a conexão de outros clientes
func testeMensagemMalformada() {
	fmt.Println("\n--- Teste 6: mensagem malformada não derruba o servidor ---")

	leitor, escritor, conexao := conectar()
	defer conexao.Close()

	msg := protocolo.Mensagem{Operacao: protocolo.OpConfirmarReserva, Payload: json.RawMessage(`{"itinerario": "isso não é uma lista"}`)}
	resp := enviarEReceber(leitor, escritor, msg)
	checar("payload malformado retorna erro sem derrubar a conexão", !resp.Sucesso, "deveria retornar erro")

	// confirma que o servidor continua respondendo normalmente depois
	leitorTeste, escritorTeste, conexaoTeste := novoCliente("passageiro_pos_malformado")
	defer conexaoTeste.Close()
	respLogin := enviarEReceber(leitorTeste, escritorTeste, protocolo.Mensagem{Operacao: protocolo.OpConsultarReservas})
	checar("servidor continua respondendo após mensagem malformada", respLogin.Sucesso, respLogin.Erro)
}

// 7. Dois motoristas, dois passageiros reservando trechos em ordens opostas: não pode travar (deadlock)
func testeDeadlockEntreMotoristasDiferentes() {
	fmt.Println("\n--- Teste 7: ausência de deadlock com locks em ordens opostas ---")

	leitorM1, escritorM1, conexaoM1 := novoCliente("motorista_A7")
	defer conexaoM1.Close()
	_, trechosA := publicarCarona(leitorM1, escritorM1, []protocolo.TrechoPedido{
		{Origem: "X7", Destino: "Y7", HorarioSaida: "2026-12-04T08:00", HorarioChegada: "2026-12-04T09:00", PrecoCentavos: 1000, AssentosTotais: 2},
	})

	leitorM2, escritorM2, conexaoM2 := novoCliente("motorista_B7")
	defer conexaoM2.Close()
	_, trechosB := publicarCarona(leitorM2, escritorM2, []protocolo.TrechoPedido{
		{Origem: "Y7", Destino: "Z7", HorarioSaida: "2026-12-04T09:00", HorarioChegada: "2026-12-04T10:00", PrecoCentavos: 1000, AssentosTotais: 2},
	})

	idA, idB := trechosA[0].ID, trechosB[0].ID

	feito := make(chan bool, 2)
	timeout := time.After(5 * time.Second)

	go func() {
		leitor, escritor, conexao := novoCliente("passageiro_ordem_AB")
		defer conexao.Close()
		confirmarReserva(leitor, escritor, []int64{idA, idB})
		feito <- true
	}()
	go func() {
		leitor, escritor, conexao := novoCliente("passageiro_ordem_BA")
		defer conexao.Close()
		confirmarReserva(leitor, escritor, []int64{idB, idA})
		feito <- true
	}()

	recebidos := 0
	for recebidos < 2 {
		select {
		case <-feito:
			recebidos++
		case <-timeout:
			checar("nenhum deadlock ao reservar trechos em ordens opostas", false, "timeout — possível deadlock")
			return
		}
	}
	checar("nenhum deadlock ao reservar trechos em ordens opostas", true, "")
}