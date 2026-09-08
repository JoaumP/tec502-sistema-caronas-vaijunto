package servidor

import (
	"encoding/json"

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
			if t.HorarioSaida[:10] != data { // compara só a parte "2006-01-02"
				continue
			}
			if t.AssentosLivres <= 0 || visitadas[t.Destino] {
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