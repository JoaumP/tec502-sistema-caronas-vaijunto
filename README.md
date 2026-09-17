# VAIJUNTO — Sistema de Caronas Compartilhadas

Trabalho individual da disciplina de Redes (TEC502) — sistema de caronas
compartilhadas de média/longa distância, com servidor central em Go
comunicando-se com os clientes via sockets TCP puros.

A especificação completa do protocolo de aplicação está em
[`PROTOCOLO.md`](./PROTOCOLO.md).

## Estrutura do projeto

```
.
├── cmd/
│   ├── servidor/            # ponto de entrada do servidor
│   ├── cliente_motorista/   # cliente de terminal para motoristas
│   ├── cliente_passageiro/  # cliente de terminal para passageiros
│   └── teste_carga/         # suíte de testes de concorrência e carga
├── internal/
│   ├── modelos/             # structs de domínio (Usuario, Trecho, Carona, Reserva)
│   ├── protocolo/           # envelope de mensagens e payloads
│   ├── tcp/                 # framing de mensagens sobre o socket
│   ├── servidor/            # estado em memória e handlers das operações
│   └── auxiliares/          # helpers de entrada, formatação de data/hora e autenticação
├── Dockerfile.servidor
├── Dockerfile.cliente_motorista
├── Dockerfile.cliente_passageiro
├── PROTOCOLO.md
└── README.md
```

## Como funciona

- O servidor mantém todo o estado (usuários, caronas, reservas) em memória,
  protegido por locks (`sync.RWMutex`/`sync.Mutex`) para acesso concorrente
  seguro. Não há banco de dados nem serviço externo de coordenação.
- A busca de itinerário trata as cidades como um grafo (trechos são arestas)
  e usa busca em profundidade (DFS) com backtracking para encontrar rotas,
  inclusive combinando caronas de motoristas diferentes.
- A reserva de um itinerário multi-trecho é atômica: todos os trechos são
  confirmados, ou nenhum é. Locks de trechos são adquiridos em ordem
  crescente de ID para evitar deadlock entre reservas concorrentes.
- Cancelar uma carona cancela em cascata as reservas afetadas e libera os
  assentos correspondentes, mesmo os pertencentes a outros motoristas no
  mesmo itinerário.

## Como executar

### Localmente (sem Docker)

Requer Go 1.26 ou superior.

```bash
# Terminal 1 — servidor (porta padrão 8080)
go run ./cmd/servidor

# Terminal 2 — cliente motorista
go run ./cmd/cliente_motorista

# Terminal 3 — cliente passageiro
go run ./cmd/cliente_passageiro
```

Endereço do servidor e porta são configuráveis por variável de ambiente:

```bash
PORTA=9000 go run ./cmd/servidor
SERVIDOR=192.168.0.10:9000 go run ./cmd/cliente_motorista
```

### Com Docker (máquinas diferentes)

Build das três imagens (na raiz do projeto):

```bash
docker build -f Dockerfile.servidor -t vaijunto-servidor .
docker build -f Dockerfile.cliente_motorista -t vaijunto-motorista .
docker build -f Dockerfile.cliente_passageiro -t vaijunto-passageiro .
```

Na máquina do servidor:

```bash
docker run -p 8080:8080 vaijunto-servidor
```

Nas máquinas dos clientes, apontando para o IP da máquina do servidor:

```bash
docker run -it -e SERVIDOR=<IP_DO_SERVIDOR>:8080 vaijunto-motorista
docker run -it -e SERVIDOR=<IP_DO_SERVIDOR>:8080 vaijunto-passageiro
```

## Testes

O diretório `cmd/teste_carga` contém uma suíte automatizada que sobe múltiplos
clientes concorrentes reais (via TCP) contra um servidor em execução,
verificando:

- que nenhum assento é vendido duas vezes sob disputa concorrente;
- que nenhum itinerário multi-trecho é confirmado parcialmente;
- que IDs de trecho duplicados no mesmo itinerário são rejeitados;
- que cancelar uma carona libera corretamente os trechos de outros
  motoristas no mesmo itinerário reservado;
- que mensagens malformadas não derrubam o servidor;
- ausência de deadlock entre reservas concorrentes com locks em ordens
  opostas;
- tempo de resposta sob carga.

Para rodar (com o servidor já em execução):

```bash
go run ./cmd/teste_carga
```

Cada execução usa um sufixo único, permitindo rodar a suíte várias vezes
sem precisar reiniciar o servidor.
