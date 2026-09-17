# Especificação do Protocolo — VAIJUNTO

## 1. Visão geral

A comunicação entre clientes (motorista e passageiro) e o servidor ocorre sobre
sockets TCP puros, sem uso de frameworks de RPC ou mensageria. As mensagens são
codificadas em **JSON** e delimitadas por um caractere de quebra de linha (`\n`).

## 2. Fluxo de conexão e desconexão

1. O cliente abre uma conexão TCP com o servidor (`net.Dial`).
2. Cliente e servidor trocam mensagens no formato definido abaixo, uma por vez
   (modelo requisição-resposta): o cliente envia uma `Mensagem`, o servidor
   responde com uma `Resposta`, e assim sucessivamente, na mesma conexão.
3. A conexão permanece aberta durante toda a sessão do usuário (podendo ele
   fazer login, executar várias operações, dar logout e logar novamente sem
   reabrir a conexão).
4. A conexão é encerrada quando o cliente fecha o programa. O servidor detecta
   o encerramento ao receber erro de leitura (EOF) no socket e libera os
   recursos daquela conexão sem afetar as demais.
5. Cada conexão é tratada por uma goroutine dedicada no servidor; um erro ou
   pânico isolado em uma conexão não derruba o servidor nem afeta outras
   conexões (uso de `recover()`).

## 3. Formato das mensagens

### 3.1 Envelope de requisição (cliente → servidor)

```json
{
  "operacao": "nome_da_operacao",
  "payload": { }
}
```

- `operacao` (string): identifica a operação solicitada.
- `payload` (objeto): dados específicos da operação, cujo formato varia
  conforme o campo `operacao` (detalhado na seção 5).

### 3.2 Envelope de resposta (servidor → cliente)

```json
{
  "sucesso": true,
  "erro": "",
  "dados": { }
}
```

- `sucesso` (bool): indica se a operação foi concluída com êxito.
- `erro` (string, opcional): mensagem de erro, presente apenas quando
  `sucesso` é `false`.
- `dados` (objeto, opcional): resultado da operação, presente apenas quando
  `sucesso` é `true` e a operação produz retorno.

### 3.3 Delimitação de mensagens

Cada mensagem JSON (requisição ou resposta) é enviada seguida de um caractere
`\n`. Como a serialização usada (`encoding/json` sem indentação) nunca insere
quebras de linha no meio do JSON, o `\n` marca de forma inequívoca o fim de
uma mensagem completa. O receptor lê o socket byte a byte até encontrar esse
delimitador antes de decodificar o JSON.

## 4. Validação e tratamento de erros

O servidor valida toda mensagem recebida antes de processá-la:

- JSON malformado ou payload com tipo incompatível ao esperado → resposta de
  erro (`sucesso: false`), sem interromper a conexão.
- Operação desconhecida → resposta de erro informando a operação inválida.
- Regras de negócio violadas (ex.: trechos com origem/destino não encadeados,
  horário de chegada anterior ao de saída, assentos inválidos, autenticação
  ausente) → resposta de erro descritiva, sem alterar o estado do servidor.

## 5. Operações

| Operação | Requer autenticação | Descrição |
|---|---|---|
| `cadastro` | não | Cria um novo usuário |
| `login` | não | Autentica um usuário existente |
| `logout` | sim | Encerra a sessão do usuário na conexão atual |
| `publicar_carona` | sim (motorista) | Publica uma nova carona com sua rota de trechos |
| `cancelar_carona` | sim (motorista) | Cancela uma carona publicada pelo próprio motorista |
| `consultar_caronas` | sim (motorista) | Lista as caronas do motorista logado, com passageiros confirmados por trecho |
| `buscar_itinerario` | não | Busca itinerários entre origem e destino numa data |
| `confirmar_reserva` | sim (passageiro) | Confirma a reserva de um itinerário (um ou mais trechos) |
| `cancelar_reserva` | sim (passageiro) | Cancela uma reserva confirmada, liberando os assentos |
| `consultar_reservas` | sim (passageiro) | Lista as reservas do passageiro logado |

### 5.1 `cadastro`

**Requisição**
```json
{
  "operacao": "cadastro",
  "payload": { "login": "joao", "senha": "123456" }
}
```

**Resposta (sucesso)**
```json
{ "sucesso": true, "dados": { "id_usuario": 1 } }
```

**Resposta (erro)**
```json
{ "sucesso": false, "erro": "login já cadastrado" }
```

### 5.2 `login`

**Requisição**
```json
{
  "operacao": "login",
  "payload": { "login": "joao", "senha": "123456" }
}
```

**Resposta**
```json
{ "sucesso": true, "dados": { "id_usuario": 1 } }
```

### 5.3 `logout`

**Requisição**
```json
{ "operacao": "logout" }
```

**Resposta**
```json
{ "sucesso": true }
```

### 5.4 `publicar_carona`

**Requisição**
```json
{
  "operacao": "publicar_carona",
  "payload": {
    "trechos": [
      {
        "origem": "Salvador",
        "destino": "Feira de Santana",
        "horario_saida": "2026-09-20T08:00",
        "horario_chegada": "2026-09-20T09:30",
        "preco_centavos": 3000,
        "assentos_totais": 3
      },
      {
        "origem": "Feira de Santana",
        "destino": "Vitória da Conquista",
        "horario_saida": "2026-09-20T09:45",
        "horario_chegada": "2026-09-20T13:00",
        "preco_centavos": 5500,
        "assentos_totais": 2
      }
    ]
  }
}
```

Regras validadas pelo servidor: origem do trecho N deve ser igual ao destino
do trecho N-1; horário de chegada deve ser posterior ao de saída; horário de
saída de um trecho não pode ser anterior ao horário de chegada do trecho
anterior; assentos totais deve ser maior que zero; preço não pode ser
negativo.

**Resposta**
```json
{ "sucesso": true, "dados": { "id_carona": 10 } }
```

### 5.5 `cancelar_carona`

**Requisição**
```json
{
  "operacao": "cancelar_carona",
  "payload": { "id_carona": 10 }
}
```

Cancela a carona e, em cascata, cancela todas as reservas de passageiros que
utilizavam algum trecho dela — liberando também os assentos de trechos de
**outros motoristas** que faziam parte do mesmo itinerário reservado.

**Resposta**
```json
{ "sucesso": true }
```

### 5.6 `consultar_caronas`

**Requisição**
```json
{ "operacao": "consultar_caronas" }
```

**Resposta**
```json
{
  "sucesso": true,
  "dados": {
    "caronas": [
      {
        "id": 10,
        "status": "ativa",
        "trechos": [
          {
            "id": 20,
            "origem": "Salvador",
            "destino": "Feira de Santana",
            "horario_saida": "2026-09-20T08:00",
            "horario_chegada": "2026-09-20T09:30",
            "assentos_totais": 3,
            "assentos_livres": 1,
            "passageiros": [5, 7]
          }
        ]
      }
    ]
  }
}
```

### 5.7 `buscar_itinerario`

**Requisição**
```json
{
  "operacao": "buscar_itinerario",
  "payload": {
    "origem": "Salvador",
    "destino": "Vitória da Conquista",
    "data": "2026-09-20"
  }
}
```

O servidor realiza uma busca em grafo (DFS com backtracking) a partir da
cidade de origem, podendo combinar trechos de motoristas diferentes,
respeitando data, assentos disponíveis, conexão de horário entre trechos e
status ativo das caronas.

**Resposta**
```json
{
  "sucesso": true,
  "dados": {
    "itinerarios": [
      {
        "trechos": [
          {
            "id": 20,
            "origem": "Salvador",
            "destino": "Feira de Santana",
            "horario_saida": "2026-09-20T08:00",
            "horario_chegada": "2026-09-20T09:30",
            "preco_centavos": 3000
          },
          {
            "id": 21,
            "origem": "Feira de Santana",
            "destino": "Vitória da Conquista",
            "horario_saida": "2026-09-20T09:45",
            "horario_chegada": "2026-09-20T13:00",
            "preco_centavos": 5500
          }
        ],
        "preco_centavos": 8500
      }
    ]
  }
}
```

Os itinerários retornados são ordenados por preço total, do mais barato ao
mais caro.

### 5.8 `confirmar_reserva`

**Requisição**
```json
{
  "operacao": "confirmar_reserva",
  "payload": { "itinerario": [20, 21] }
}
```

A confirmação é atômica: todos os trechos do itinerário são reservados, ou
nenhum é. Trechos são travados (mutex individual) em ordem crescente de ID
para evitar deadlock entre reservas concorrentes que disputam os mesmos
trechos em ordens diferentes. IDs repetidos no mesmo itinerário e trechos
pertencentes a caronas já canceladas são rejeitados.

**Resposta (sucesso)**
```json
{ "sucesso": true, "dados": { "id_reserva": 8 } }
```

**Resposta (erro — assento indisponível)**
```json
{ "sucesso": false, "erro": "assento indisponível em um dos trechos" }
```

### 5.9 `cancelar_reserva`

**Requisição**
```json
{
  "operacao": "cancelar_reserva",
  "payload": { "id_reserva": 8 }
}
```

**Resposta**
```json
{ "sucesso": true }
```

### 5.10 `consultar_reservas`

**Requisição**
```json
{ "operacao": "consultar_reservas" }
```

**Resposta**
```json
{
  "sucesso": true,
  "dados": {
    "reservas": [
      {
        "id": 8,
        "status": "confirmada",
        "trechos": [
          {
            "id": 20,
            "origem": "Salvador",
            "destino": "Feira de Santana",
            "horario_saida": "2026-09-20T08:00",
            "horario_chegada": "2026-09-20T09:30",
            "preco_centavos": 3000
          }
        ]
      }
    ]
  }
}
```

## 6. Exemplo de troca completa de mensagens

```
Cliente → Servidor: {"operacao":"login","payload":{"login":"joao","senha":"123456"}}\n
Servidor → Cliente: {"sucesso":true,"dados":{"id_usuario":1}}\n

Cliente → Servidor: {"operacao":"buscar_itinerario","payload":{"origem":"Salvador","destino":"Feira de Santana","data":"2026-09-20"}}\n
Servidor → Cliente: {"sucesso":true,"dados":{"itinerarios":[{"trechos":[{"id":20,"origem":"Salvador","destino":"Feira de Santana","horario_saida":"2026-09-20T08:00","horario_chegada":"2026-09-20T09:30","preco_centavos":3000}],"preco_centavos":3000}]}}\n

Cliente → Servidor: {"operacao":"confirmar_reserva","payload":{"itinerario":[20]}}\n
Servidor → Cliente: {"sucesso":true,"dados":{"id_reserva":8}}\n
```
