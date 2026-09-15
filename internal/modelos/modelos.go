package modelos

import (
	"sync"
	"time"
)

// Usuario: quem usa o sistema (motorista ou passageiro)
type Usuario struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	SenhaHash string `json:"senha_hash"`
}

// Trecho: menor unidade de viagem, entre duas cidades.
type Trecho struct {
	ID       int64 `json:"id"`
	IDCarona int64 `json:"id_carona"`

	Origem  string `json:"origem"`
	Destino string `json:"destino"`

	HorarioSaida   string `json:"horario_saida"`	// formato "AAAA-MM-DDTHH:mm", ex. "2024-06-15T14:30"
	HorarioChegada string `json:"horario_chegada"`

	PrecoCentavos int `json:"preco_centavos"`

	AssentosTotais int `json:"assentos_totais"`
	AssentosLivres int `json:"assentos_livres"`

	mu sync.Mutex `json:"-"` // protege AssentosLivres; json:"-" evita tentar serializar o mutex
}

// TentarReservar tenta ocupar 1 assento. Retorna false se não há vaga.
func (t *Trecho) TentarReservar() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.AssentosLivres <= 0 {
		return false
	}
	t.AssentosLivres--
	return true
}

// Liberar devolve 1 assento (usado se a reserva atômica falhar em outro trecho)
func (t *Trecho) Liberar() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.AssentosLivres++
}

// Carona: viagem anunciada pelo motorista, com sua rota de trechos.
// Trechos guarda ponteiros para os mesmos objetos usados no indice de busca,
// assim os dois lugares sempre veem o estado atualizado.
type Carona struct {
	ID          int64 `json:"id"`
	MotoristaID int64 `json:"motorista_id"`
	Status string `json:"status"` // "ativa" ou "cancelada"
	Trechos []*Trecho `json:"trechos"`
}

// Reserva: bilhete do passageiro, um itinerário de um ou mais trechos
type Reserva struct {
	ID           int64 `json:"id"`
	PassageiroID int64 `json:"passageiro_id"`

	Itinerario []int64 `json:"itinerario"` // IDs de Trecho, em ordem de viagem

	Status string `json:"status"` // "confirmada", "cancelada"

	CriadoEm time.Time `json:"criado_em"`
}
