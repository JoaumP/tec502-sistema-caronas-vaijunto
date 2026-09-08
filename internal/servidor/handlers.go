package servidor

import (
	"encoding/json"
	"sort"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/modelos"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/protocolo"
)


// respostaErro: helper pra montar uma Resposta de erro rapidinho
func respostaErro(msg string) protocolo.Resposta {
	return protocolo.Resposta{Sucesso: false, Erro: msg}
}


func (e *Estado) HandleLogin(payload json.RawMessage) protocolo.Resposta {
	var pedido protocolo.PedidoLogin
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	usuario, existe := e.BuscarUsuario(pedido.Login)

	if existe {
		// usuário já existe: valida a senha
		err := bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(pedido.Senha))
		if err != nil {
			return respostaErro("senha incorreta")
		}
	} else {
		// usuário novo: cadastra
		hash, err := bcrypt.GenerateFromPassword([]byte(pedido.Senha), bcrypt.DefaultCost)
		if err != nil {
			return respostaErro("erro ao processar senha")
		}
		usuario = &modelos.Usuario{
			ID:        e.GerarIDUsuario(),
			Login:     pedido.Login,
			SenhaHash: string(hash),
		}
		e.SalvarUsuario(usuario)
	}

	dados, _ := json.Marshal(protocolo.RespostaLogin{IDUsuario: usuario.ID})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}

func (e *Estado) HandlePublicarCarona(idMotorista int64, payload json.RawMessage) protocolo.Resposta {
	if idMotorista == 0 {
		return respostaErro("não autenticado")
	}

	var pedido protocolo.PedidoPublicarCarona
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	if len(pedido.Trechos) == 0 {
		return respostaErro("carona precisa de pelo menos um trecho")
	}

	idCarona := e.GerarIDCarona()

	var trechos []*modelos.Trecho
	for _, tp := range pedido.Trechos {
		trecho := &modelos.Trecho{
			ID:             e.GerarIDTrecho(),
			IDCarona:       idCarona,
			Origem:         tp.Origem,
			Destino:        tp.Destino,
			HorarioSaida:   tp.HorarioSaida,
			HorarioChegada: tp.HorarioChegada,
			PrecoCentavos:  tp.PrecoCentavos,
			AssentosTotais: tp.AssentosTotais,
			AssentosLivres: tp.AssentosTotais,
		}
		trechos = append(trechos, trecho)
	}

	carona := &modelos.Carona{
		ID:           idCarona,
		MotoristaID:  idMotorista,
		Data:         pedido.Data,
		HorarioSaida: pedido.HorarioSaida,
		Status:       "ativa",
		Trechos:      trechos,
	}

	e.SalvarCarona(carona)

	dados, _ := json.Marshal(protocolo.RespostaPublicarCarona{IDCarona: carona.ID})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}

func (e *Estado) HandleBuscarItinerario(payload json.RawMessage) protocolo.Resposta {
	var pedido protocolo.PedidoBuscarItinerario
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	caminhos := e.buscarCaminhos(pedido.Origem, pedido.Destino, pedido.Data)

	var itinerarios []protocolo.ItinerarioEncontrado
	for _, caminho := range caminhos {
		var ids []int64
		preco := 0
		for _, t := range caminho {
			ids = append(ids, t.ID)
			preco += t.PrecoCentavos
		}
		itinerarios = append(itinerarios, protocolo.ItinerarioEncontrado{
			Trechos:       ids,
			PrecoCentavos: preco,
		})
	}

	dados, _ := json.Marshal(protocolo.RespostaBuscarItinerario{Itinerarios: itinerarios})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}

// buscarCaminhos: DFS com backtracking a partir da origem
func (e *Estado) buscarCaminhos(origem, destino, data string) [][]*modelos.Trecho {
	var resultados [][]*modelos.Trecho
	var caminhoAtual []*modelos.Trecho
	visitadas := make(map[string]bool)

	var dfs func(cidade string)
	dfs = func(cidade string) {
		if cidade == destino {
			copia := make([]*modelos.Trecho, len(caminhoAtual))
			copy(copia, caminhoAtual)
			resultados = append(resultados, copia)
			return
		}
		visitadas[cidade] = true
		for _, t := range e.TrechosSaindoDe(cidade) {
			if t.HorarioSaida[:10] != data {
				continue
			}
			if t.AssentosLivres <= 0 || visitadas[t.Destino] {
				continue
			}

			carona, existe := e.BuscarCarona(t.IDCarona)
			if !existe || carona.Status != "ativa" {
				continue
			}

			caminhoAtual = append(caminhoAtual, t)
			dfs(t.Destino)
			caminhoAtual = caminhoAtual[:len(caminhoAtual)-1]
		}
		visitadas[cidade] = false
	}

	dfs(origem)
	return resultados
}

func (e *Estado) HandleConfirmarReserva(idPassageiro int64, payload json.RawMessage) protocolo.Resposta {
	if idPassageiro == 0 {
		return respostaErro("não autenticado")
	}

	var pedido protocolo.PedidoConfirmarReserva
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	if len(pedido.Itinerario) == 0 {
		return respostaErro("itinerário vazio")
	}

	// busca os trechos reais a partir dos IDs
	trechos := make([]*modelos.Trecho, 0, len(pedido.Itinerario))
	for _, idTrecho := range pedido.Itinerario {
		t := e.BuscarTrecho(idTrecho)
		if t == nil {
			return respostaErro("trecho não encontrado")
		}
		trechos = append(trechos, t)
	}

	// ordena por ID para sempre travar na mesma ordem (evita deadlock)
	sort.Slice(trechos, func(i, j int) bool {
		return trechos[i].ID < trechos[j].ID
	})

	// tenta reservar todos, na ordem
	var reservados []*modelos.Trecho
	for _, t := range trechos {
		if !t.TentarReservar() {
			// falhou: desfaz tudo que já reservou até agora
			for _, r := range reservados {
				r.Liberar()
			}
			return respostaErro("assento indisponível em um dos trechos")
		}
		reservados = append(reservados, t)
	}

	// tudo certo: monta e salva a reserva
	reserva := &modelos.Reserva{
		ID:           e.GerarIDReserva(),
		PassageiroID: idPassageiro,
		Itinerario:   pedido.Itinerario,
		Status:       "confirmada",
		CriadoEm:     time.Now(),
	}
	e.SalvarReserva(reserva)

	dados, _ := json.Marshal(protocolo.RespostaConfirmarReserva{IDReserva: reserva.ID})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}

func (e *Estado) HandleCancelarCarona(idMotorista int64, payload json.RawMessage) protocolo.Resposta {
	if idMotorista == 0 {
		return respostaErro("não autenticado")
	}

	var pedido protocolo.PedidoCancelarCarona
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	carona, existe := e.BuscarCarona(pedido.IDCarona)
	if !existe {
		return respostaErro("carona não encontrada")
	}
	if carona.MotoristaID != idMotorista {
		return respostaErro("carona não pertence a este motorista")
	}

	carona.Status = "cancelada"

	return protocolo.Resposta{Sucesso: true}
}

func (e *Estado) HandleConsultarCaronas(idMotorista int64) protocolo.Resposta {
	if idMotorista == 0 {
		return respostaErro("não autenticado")
	}

	var resultado []protocolo.CaronaDetalhada
	for _, c := range e.TodasCaronas() {
		if c.MotoristaID != idMotorista {
			continue
		}

		var trechos []protocolo.TrechoDetalhado
		for _, t := range c.Trechos {
			trechos = append(trechos, protocolo.TrechoDetalhado{
				ID:             t.ID,
				Origem:         t.Origem,
				Destino:        t.Destino,
				AssentosTotais: t.AssentosTotais,
				AssentosLivres: t.AssentosLivres,
				Passageiros:    e.PassageirosDoTrecho(t.ID),
			})
		}

		resultado = append(resultado, protocolo.CaronaDetalhada{
			ID:           c.ID,
			Data:         c.Data,
			HorarioSaida: c.HorarioSaida,
			Status:       c.Status,
			Trechos:      trechos,
		})
	}

	dados, _ := json.Marshal(protocolo.RespostaConsultarCaronas{Caronas: resultado})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}

func (e *Estado) HandleCancelarReserva(idPassageiro int64, payload json.RawMessage) protocolo.Resposta {
	if idPassageiro == 0 {
		return respostaErro("não autenticado")
	}

	var pedido protocolo.PedidoCancelarReserva
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	reserva, existe := e.BuscarReserva(pedido.IDReserva)
	if !existe {
		return respostaErro("reserva não encontrada")
	}
	if reserva.PassageiroID != idPassageiro {
		return respostaErro("reserva não pertence a este passageiro")
	}
	if reserva.Status != "confirmada" {
		return respostaErro("reserva já não está mais confirmada")
	}

	// devolve os assentos
	for _, idTrecho := range reserva.Itinerario {
		if t := e.BuscarTrecho(idTrecho); t != nil {
			t.Liberar()
		}
	}
	reserva.Status = "cancelada"

	return protocolo.Resposta{Sucesso: true}
}

func (e *Estado) HandleConsultarReservas(idPassageiro int64) protocolo.Resposta {
	if idPassageiro == 0 {
		return respostaErro("não autenticado")
	}

	var resultado []protocolo.ReservaDetalhada
	for _, r := range e.TodasReservas() {
		if r.PassageiroID != idPassageiro {
			continue
		}
		resultado = append(resultado, protocolo.ReservaDetalhada{
			ID:         r.ID,
			Itinerario: r.Itinerario,
			Status:     r.Status,
		})
	}

	dados, _ := json.Marshal(protocolo.RespostaConsultarReservas{Reservas: resultado})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}

