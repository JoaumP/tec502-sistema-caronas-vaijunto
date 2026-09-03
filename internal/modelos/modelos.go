package modelos

import "time"

// Usuario: quem usa o sistema (motorista ou passageiro)
type Usuario struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	SenhaHash string `json:"senha_hash"`
}

// Trecho: menor unidade de viagem, entre duas cidades.
// Horarios ja vem calculados pelo servidor a partir da entrada do motorista.
type Trecho struct {
	ID       int64 `json:"id"`
	IDCarona int64 `json:"id_carona"`

	Origem  string `json:"origem"`
	Destino string `json:"destino"`

	HorarioSaida   string `json:"horario_saida"`   // formato "2006-01-02T15:04"
	HorarioChegada string `json:"horario_chegada"` // formato "2006-01-02T15:04"

	PrecoCentavos int `json:"preco_centavos"`

	AssentosTotais int `json:"assentos_totais"`
	AssentosLivres int `json:"assentos_livres"` // protegido por mutex no servidor
}

// Carona: viagem anunciada pelo motorista, com sua rota de trechos.
// Trechos guarda ponteiros para os mesmos objetos usados no indice de busca,
// assim os dois lugares sempre veem o estado atualizado.
type Carona struct {
	ID          int64 `json:"id"`
	MotoristaID int64 `json:"motorista_id"`

	Data         string `json:"data"`          // formato "2006-01-02"
	HorarioSaida string `json:"horario_saida"` // formato "2006-01-02T15:04"

	Status string `json:"status"` // "ativa" ou "cancelada"

	Trechos []*Trecho `json:"trechos"`
}

// Reserva: bilhete do passageiro, um itinerário de um ou mais trechos
type Reserva struct {
	ID           int64 `json:"id"`
	PassageiroID int64 `json:"passageiro_id"`

	Itinerario []int64 `json:"itinerario"` // IDs de Trecho, em ordem de viagem

	Status string `json:"status"` // "pendente", "confirmada", "cancelada" ou "expirada"

	CriadoEm time.Time `json:"criado_em"`
}

// TrechoEntrada: usada só no momento de publicar a carona, não é salva.
// O motorista informa duracao/espera; o servidor calcula os horarios do Trecho.
type TrechoEntrada struct {
	Origem         string `json:"origem"`
	Destino        string `json:"destino"`
	DuracaoMinutos int    `json:"duracao_minutos"`
	EsperaMinutos  int    `json:"espera_minutos"` // opcional, padrão 0
	PrecoCentavos  int    `json:"preco_centavos"`
	AssentosTotais int    `json:"assentos_totais"`
}