package protocolo

import (
	"encoding/json"
)

// Mensagem: envelope padrão de toda comunicação cliente -> servidor
type Mensagem struct {
	Operacao string          `json:"operacao"`
	Payload  json.RawMessage `json:"payload"`
}

// Resposta: envelope padrão de retorno do servidor -> cliente
type Resposta struct {
	Sucesso bool            `json:"sucesso"`
	Erro    string          `json:"erro,omitempty"`
	Dados   json.RawMessage `json:"dados,omitempty"`
}

// Lista de operações suportadas
const (
	OpLogin             = "login"
	OpPublicarCarona    = "publicar_carona"
	OpCancelarCarona    = "cancelar_carona"
	OpConsultarCaronas  = "consultar_caronas"
	OpBuscarItinerario  = "buscar_itinerario"
	OpConfirmarReserva  = "confirmar_reserva"
	OpCancelarReserva   = "cancelar_reserva"
	OpConsultarReservas = "consultar_reservas"
)

// --- Payloads de entrada (cliente -> servidor) ---

type PedidoLogin struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
}

// PedidoPublicarCarona: rota completa, motorista informa cada trecho já com horários
type PedidoPublicarCarona struct {
	Data         string           `json:"data"`
	HorarioSaida string           `json:"horario_saida"`
	Trechos      []TrechoPedido   `json:"trechos"`
}

// TrechoPedido: um trecho da rota, como o motorista informa ao publicar
type TrechoPedido struct {
	Origem         string `json:"origem"`
	Destino        string `json:"destino"`
	HorarioSaida   string `json:"horario_saida"`
	HorarioChegada string `json:"horario_chegada"`
	PrecoCentavos  int    `json:"preco_centavos"`
	AssentosTotais int    `json:"assentos_totais"`
}

type PedidoCancelarCarona struct {
	IDCarona int64 `json:"id_carona"`
}

// PedidoConsultarCaronas: sem filtro, lista as caronas do motorista logado
type PedidoConsultarCaronas struct{}

type PedidoBuscarItinerario struct {
	Origem  string `json:"origem"`
	Destino string `json:"destino"`
	Data    string `json:"data"`
}

// PedidoConfirmarReserva: lista de IDs de trecho escolhidos pelo passageiro
type PedidoConfirmarReserva struct {
	Itinerario []int64 `json:"itinerario"`
}

type PedidoCancelarReserva struct {
	IDReserva int64 `json:"id_reserva"`
}

// PedidoConsultarReservas: sem filtro, lista as reservas do passageiro logado
type PedidoConsultarReservas struct{}

// --- Payloads de saída (servidor -> cliente, vão dentro de Resposta.Dados) ---

type RespostaLogin struct {
	IDUsuario int64 `json:"id_usuario"`
}

type RespostaPublicarCarona struct {
	IDCarona int64 `json:"id_carona"`
}

type RespostaBuscarItinerario struct {
	Itinerarios []ItinerarioEncontrado `json:"itinerarios"`
}

// ItinerarioEncontrado: uma opção de viagem, com o preço já somado
type ItinerarioEncontrado struct {
	Trechos       []int64 `json:"trechos"` // IDs, na ordem da viagem
	PrecoCentavos int     `json:"preco_centavos"`
}

type RespostaConfirmarReserva struct {
	IDReserva int64 `json:"id_reserva"`
}