package protocolo

import "encoding/json"

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
	OpLogin              = "login"
	OpPublicarCarona     = "publicar_carona"
	OpCancelarCarona     = "cancelar_carona"
	OpConsultarCaronas   = "consultar_caronas"
	OpBuscarItinerario   = "buscar_itinerario"
	OpConfirmarReserva   = "confirmar_reserva"
	OpCancelarReserva    = "cancelar_reserva"
	OpConsultarReservas  = "consultar_reservas"
)