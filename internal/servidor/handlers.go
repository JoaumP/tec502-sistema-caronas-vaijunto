package servidor

import (
	"encoding/json"
	"sort"
	"time"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/modelos"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/protocolo"
)


// respostaErro: helper pra montar uma Resposta de erro rapidinho
func respostaErro(msg string) protocolo.Resposta {
	return protocolo.Resposta{Sucesso: false, Erro: msg}
}


func (e *Estado) HandleCadastro(payload json.RawMessage) protocolo.Resposta {
	var pedido protocolo.PedidoLogin
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pedido.Senha), bcrypt.DefaultCost)
	if err != nil {
		return respostaErro("erro ao processar senha")
	}

	usuario, criado := e.CriarUsuario(pedido.Login, string(hash))
	if !criado {
		return respostaErro("login já cadastrado")
	}

	dados, _ := json.Marshal(protocolo.RespostaLogin{IDUsuario: usuario.ID})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}

func (e *Estado) HandleLogin(payload json.RawMessage) protocolo.Resposta {
	var pedido protocolo.PedidoLogin
	if err := json.Unmarshal(payload, &pedido); err != nil {
		return respostaErro("payload inválido")
	}

	usuario, existe := e.BuscarUsuario(pedido.Login)
	if !existe {
		return respostaErro("login não encontrado")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(pedido.Senha)); err != nil {
		return respostaErro("senha incorreta")
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

	if err := validarTrechos(pedido.Trechos); err != nil {
		return respostaErro(err.Error())
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
		var trechos []protocolo.TrechoResumo
		preco := 0
		for _, t := range caminho {
			trechos = append(trechos, protocolo.TrechoResumo{
				ID:             t.ID,
				Origem:         t.Origem,
				Destino:        t.Destino,
				HorarioSaida:   t.HorarioSaida,
				HorarioChegada: t.HorarioChegada,
				PrecoCentavos:  t.PrecoCentavos,
			})
			preco += t.PrecoCentavos
		}
		itinerarios = append(itinerarios, protocolo.ItinerarioEncontrado{
			Trechos:       trechos,
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

	var dfs func(cidade string, chegadaAnterior string)
	dfs = func(cidade string, chegadaAnterior string) {
		if cidade == destino {
			copia := make([]*modelos.Trecho, len(caminhoAtual))
			copy(copia, caminhoAtual)
			resultados = append(resultados, copia)
			return
		}
		visitadas[cidade] = true
		for _, t := range e.TrechosSaindoDe(cidade) {
			if len(t.HorarioSaida) < 10 {
				continue
			}

			if chegadaAnterior == "" {
				// primeiro trecho: precisa sair na data pedida
				if t.HorarioSaida[:10] != data {
					continue
				}
			} else if t.HorarioSaida < chegadaAnterior {
				// trechos seguintes: só pode sair depois da chegada anterior
				continue
			}

			if t.AssentosLivres <= 0 || visitadas[t.Destino] {
				continue
			}

			carona, existe := e.BuscarCarona(t.IDCarona)
			if !existe || !carona.EstaAtiva() {
				continue
			}

			caminhoAtual = append(caminhoAtual, t)
			dfs(t.Destino, t.HorarioChegada)
			caminhoAtual = caminhoAtual[:len(caminhoAtual)-1]
		}
		visitadas[cidade] = false
	}

	dfs(origem, "")
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
		carona, existe := e.BuscarCarona(t.IDCarona)
		if !existe || !carona.EstaAtiva() {
			return respostaErro("um dos trechos pertence a uma carona cancelada")
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
	if !carona.EstaAtiva() {
		return respostaErro("carona já está cancelada")
	}

	carona.Cancelar()
	e.CancelarReservasDaCarona(carona)

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
				HorarioSaida:   t.HorarioSaida,
				HorarioChegada: t.HorarioChegada,
				AssentosTotais: t.AssentosTotais,
				AssentosLivres: t.AssentosLivres,
				Passageiros:    e.PassageirosDoTrecho(t.ID),
			})
		}

		resultado = append(resultado, protocolo.CaronaDetalhada{
			ID:           c.ID,
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
	reserva.Cancelar()

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

		var trechos []protocolo.TrechoResumo
		for _, idTrecho := range r.Itinerario {
			t := e.BuscarTrecho(idTrecho)
			if t == nil {
				continue
			}
			trechos = append(trechos, protocolo.TrechoResumo{
				ID:             t.ID,
				Origem:         t.Origem,
				Destino:        t.Destino,
				HorarioSaida:   t.HorarioSaida,
				HorarioChegada: t.HorarioChegada,
				PrecoCentavos:  t.PrecoCentavos,
			})
		}

		resultado = append(resultado, protocolo.ReservaDetalhada{
			ID:      r.ID,
			Trechos: trechos,
			Status:  r.StatusAtual(),
		})
	}

	dados, _ := json.Marshal(protocolo.RespostaConsultarReservas{Reservas: resultado})
	return protocolo.Resposta{Sucesso: true, Dados: dados}
}



func validarTrechos(trechos []protocolo.TrechoPedido) error {
	if len(trechos) == 0 {
		return errors.New("carona precisa de pelo menos um trecho")
	}
	for i, tp := range trechos {
		if tp.Origem == "" || tp.Destino == "" {
			return errors.New("origem e destino não podem ser vazios")
		}
		if len(tp.HorarioSaida) < 10 || len(tp.HorarioChegada) < 10 {
			return errors.New("horário inválido")
		}
		if tp.HorarioChegada <= tp.HorarioSaida {
			return errors.New("horário de chegada deve ser depois da saída")
		}
		if tp.AssentosTotais <= 0 {
			return errors.New("assentos totais deve ser maior que zero")
		}
		if tp.PrecoCentavos < 0 {
			return errors.New("preço não pode ser negativo")
		}
		if i > 0 && tp.Origem != trechos[i-1].Destino {
			return errors.New("origem deve ser o destino do trecho anterior")
		}
		if i > 0 && tp.HorarioSaida < trechos[i-1].HorarioChegada {
			return errors.New("um trecho não pode partir antes do anterior chegar")
		}
	}
	return nil
}