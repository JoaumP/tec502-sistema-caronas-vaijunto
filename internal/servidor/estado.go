package servidor

import (
	"sync"
	"sync/atomic"
	"github.com/JoaumP/tec502-sistema-caronas-vaijunto/internal/modelos"
)

// Estado: tudo que o servidor guarda em memória enquanto roda
type Estado struct {
	usuariosMu sync.RWMutex
	usuarios   map[string]*modelos.Usuario // chave: login

	caronasMu sync.RWMutex
	caronas   map[int64]*modelos.Carona // chave: ID da carona

	trechosPorOrigemMu sync.RWMutex
	trechosPorOrigem   map[string][]*modelos.Trecho // chave: cidade de origem

	reservasMu sync.RWMutex
	reservas   map[int64]*modelos.Reserva // chave: ID da reserva

	// contadores atômicos de ID, um por tipo de entidade
	proximoIDUsuario int64
	proximoIDCarona  int64
	proximoIDTrecho  int64
	proximoIDReserva int64
}

func NovoEstado() *Estado {
	return &Estado{
		usuarios:         make(map[string]*modelos.Usuario),
		caronas:          make(map[int64]*modelos.Carona),
		trechosPorOrigem: make(map[string][]*modelos.Trecho),
		reservas:         make(map[int64]*modelos.Reserva),
	}
}

// Geração de IDs

func (e *Estado) GerarIDUsuario() int64 { return atomic.AddInt64(&e.proximoIDUsuario, 1) }
func (e *Estado) GerarIDCarona() int64  { return atomic.AddInt64(&e.proximoIDCarona, 1) }
func (e *Estado) GerarIDTrecho() int64  { return atomic.AddInt64(&e.proximoIDTrecho, 1) }
func (e *Estado) GerarIDReserva() int64 { return atomic.AddInt64(&e.proximoIDReserva, 1) }

// Usuários

func (e *Estado) SalvarUsuario(u *modelos.Usuario) {
	e.usuariosMu.Lock()
	defer e.usuariosMu.Unlock()
	e.usuarios[u.Login] = u
}

func (e *Estado) BuscarUsuario(login string) (*modelos.Usuario, bool) {
	e.usuariosMu.RLock()
	defer e.usuariosMu.RUnlock()
	u, existe := e.usuarios[login]
	return u, existe
}

// Caronas

func (e *Estado) SalvarCarona(c *modelos.Carona) {
	e.caronasMu.Lock()
	e.caronas[c.ID] = c
	e.caronasMu.Unlock()

	e.trechosPorOrigemMu.Lock()
	defer e.trechosPorOrigemMu.Unlock()
	for _, t := range c.Trechos {
		e.trechosPorOrigem[t.Origem] = append(e.trechosPorOrigem[t.Origem], t)
	}
}

func (e *Estado) BuscarCarona(id int64) (*modelos.Carona, bool) {
	e.caronasMu.RLock()
	defer e.caronasMu.RUnlock()
	c, existe := e.caronas[id]
	return c, existe
}

// Trechos (para busca de itinerário)

func (e *Estado) TrechosSaindoDe(cidade string) []*modelos.Trecho {
	e.trechosPorOrigemMu.RLock()
	defer e.trechosPorOrigemMu.RUnlock()
	return e.trechosPorOrigem[cidade]
}

// Reservas

func (e *Estado) SalvarReserva(r *modelos.Reserva) {
	e.reservasMu.Lock()
	defer e.reservasMu.Unlock()
	e.reservas[r.ID] = r
}

func (e *Estado) BuscarReserva(id int64) (*modelos.Reserva, bool) {
	e.reservasMu.RLock()
	defer e.reservasMu.RUnlock()
	r, existe := e.reservas[id]
	return r, existe
}