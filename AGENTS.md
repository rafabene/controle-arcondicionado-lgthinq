# AGENTS.md

## Visão geral

Binário único em Go (`cmd/economizador`) que assina eventos MQTT da LG ThinQ e eleva `temperature.targetTemperature` para `MIN_TEMPERATURE` quando o aparelho está ligado e abaixo do mínimo.

- `cmd/economizador/main.go` — fluxo do programa e regra de ajuste (`enforceMinTemperature`).
- `internal/thinq/` — HTTP, mTLS/MQTT e parsing de estado (`state.go`).
- `internal/config/` — variáveis de ambiente.
- Alvo de execução real é um Raspberry Pi: `make build-pi` cross-compila `linux/arm64`.

## Comandos

```bash
make build      # bin/economizador
make run        # build + executa
make build-pi   # bin/economizador-pi (linux/arm64)

go build ./... && go vet ./... && go test ./...
go test ./internal/thinq -run TestPowerOn -v   # teste único
go run golang.org/x/vuln/cmd/govulncheck@latest ./...   # scan de vulnerabilidades
```

Não há CI, lint, pre-commit nem opencode.json: a verificação é `build + vet + test`.

O `govulncheck` instalado em `~/go/bin` foi compilado com Go antigo e falha com a stdlib atual; rode sempre via `go run ...@latest`.

## Execução e configuração

- `config.Load()` usa `godotenv.Load()`, que lê `.env` **do diretório atual** e falha se o arquivo não existir. Rodar de fora da raiz do repositório quebra; e não há modo mock — executar a aplicação com `THINQ_PAT` válido envia comandos reais aos aparelhos.
- `MIN_TEMPERATURE` é forçado para no mínimo `21` em `internal/config/config.go`; valores menores em `.env` são elevados silenciosamente.
- Sem `THINQ_CLIENT_ID`, cada execução gera um client ID novo (`go-thinq-<uuid>`) e registra outro cliente MQTT/Thing. Fixe o valor no `.env` para reutilizar o mesmo cliente entre execuções.
- O log sai no stdout **e** em `economizador.log` na raiz (gitignored). É a fonte de verdade para confirmar o que aconteceu em execução.

## Quirks da API ThinQ

- Base `https://api-aic.lgthinq.com`; os headers obrigatórios (incluindo `x-api-key` fixo e `x-country`) são montados em `setHeaders`.
- Comando de temperatura: `POST /devices/{id}/control` com `{"temperature":{"targetTemperature":N}}`, **sem** wrapper `dataSetList`, e com o header `x-conditional-control: true`.
- O `report` de um push e o `response` de `GET /devices/{id}/state` têm o mesmo formato; `internal/thinq/state.go` lê os dois com as mesmas funções.
- Pushes são parciais: em geral trazem só `temperature.currentTemperature`, sem `targetTemperature` nem `operation`. Nunca assuma que um push carrega o estado completo; quando o estado de energia é desconhecido, é preciso consultar `GET /devices/{id}/state`.
- Power fica em `operation.airConOperationMode` (`POWER_ON` / `POWER_OFF`). Enviar temperatura com o aparelho em `POWER_OFF` retorna 400 com code `2304` (`Command not supported in POWER OFF`); por isso o ajuste ignora aparelhos desligados.
- Inscrição duplicada conta como sucesso: HTTP 409, ou 404 com `error.code == "1207"` (`Already subscribed push`).
- A temperatura vem na unidade do aparelho (`temperature.unit`), mas o código compara e envia sempre como Celsius — não existe conversão de Fahrenheit.

## Convenções

- Código, nomes e comentários em inglês; log e documentação em português. Linhas de log de ajuste começam com `[<alias>]`.
- A regra de ajuste chama o cliente HTTP diretamente, então não tem teste; o parsing puro fica em `internal/thinq/state.go` e é coberto por `state_test.go` — testes novos de lógica devem ir por esse caminho.
- `gofmt -l .` já acusa `internal/thinq/client.go` e `internal/thinq/mqtt_models.go` antes de qualquer alteração. Não reformate arquivos fora do escopo da tarefa.
- `go.mod` marca todas as dependências como `// indirect`, mesmo as importadas diretamente. Não rode `go mod tidy` como efeito colateral de outra mudança.
- Commits em português, no imperativo, sem atribuição de ferramenta.
