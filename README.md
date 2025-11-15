# Economizador de Energia - LG ThinQ

Aplicação Go que monitora e controla automaticamente a temperatura de ar condicionados LG através da API ThinQ, mantendo sempre em 24°C para economia de energia.

## Estrutura do Projeto

```
controle-arcondicionado/
├── cmd/
│   └── economizador/        # Aplicação economizador
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go        # Configuração e carregamento de .env
│   └── thinq/
│       ├── client.go        # Cliente HTTP e controle da API ThinQ
│       ├── models.go        # Modelos de dados da API
│       └── mqtt_models.go   # Modelos de dados do MQTT
├── .env                     # Variáveis de ambiente (não versionado)
├── .env.example             # Exemplo de configuração (versionado)
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Funcionalidades

- Conexão em tempo real via MQTT com dispositivos LG ThinQ
- Detecção automática de mudanças de temperatura
- Ajuste imediato da temperatura para 24°C
- Suporte para múltiplos dispositivos simultaneamente
- Sem cooldown - ajusta sempre que detecta mudança

## Requisitos

- Go 1.21 ou superior
- Conta LG ThinQ com dispositivos cadastrados
- Personal Access Token (PAT) da LG ThinQ

## Como obter o Personal Access Token (PAT)

1. Acesse https://connect-pat.lgthinq.com/
2. Faça login com sua conta LG ThinQ
3. Copie o token gerado
4. Cole no arquivo `.env`

## Instalação

1. Clone o repositório (ou baixe o código)

2. Instale as dependências:
```bash
go mod download
```

3. Configure o arquivo `.env`:
```bash
cp .env.example .env
```

Edite o arquivo `.env` e adicione suas credenciais:
```env
THINQ_PAT=seu_token_aqui
THINQ_COUNTRY_CODE=BR
```

**Nota:** O Client ID é gerado automaticamente pela aplicação, não precisa configurar.

## Uso

Para iniciar o economizador de energia:

```bash
go run cmd/economizador/main.go
```

O sistema irá:
1. Conectar aos servidores LG ThinQ
2. Listar todos os dispositivos encontrados
3. Inscrever-se para receber eventos de cada dispositivo
4. Monitorar mudanças de temperatura em tempo real
5. Ajustar automaticamente para 24°C sempre que detectar uma mudança

Para parar o programa, pressione `Ctrl+C`.

## Como Funciona

1. **Autenticação**: Usa o Personal Access Token para autenticar com a API LG ThinQ
2. **Descoberta**: Lista todos os dispositivos de ar condicionado na conta
3. **Registro MQTT**: Registra o cliente e obtém certificados para conexão MQTT segura (mTLS)
4. **Inscrição**: Inscreve-se para receber eventos de cada dispositivo
5. **Monitoramento**: Escuta eventos MQTT em tempo real
6. **Controle**: Quando detecta `targetTemperature` diferente de 24°C, envia comando para ajustar

## Configuração Avançada

### Alterando a Temperatura Alvo

Para mudar de 24°C para outra temperatura, edite `cmd/economizador/main.go`:

```go
const (
    targetTemperature = 24  // Altere para a temperatura desejada
)
```

## API Endpoints Utilizados

- `GET /devices` - Lista dispositivos
- `GET /route` - Obtém servidor MQTT
- `POST /client` - Registra cliente MQTT
- `POST /client/certificate` - Obtém certificado para MQTT
- `POST /event/{deviceId}/subscribe` - Inscreve para eventos do dispositivo
- `POST /push/{deviceId}/subscribe` - Inscreve para notificações push
- `POST /devices/{deviceId}/control` - Controla temperatura do dispositivo

## Tecnologias Utilizadas

- **Go**: Linguagem de programação
- **Paho MQTT**: Cliente MQTT para comunicação em tempo real
- **godotenv**: Gerenciamento de variáveis de ambiente
- **LG ThinQ Connect API**: API oficial LG para IoT

## Troubleshooting

### "Failed to get MQTT route"
- Verifique se o PAT está correto
- Confirme que o país está configurado corretamente (BR para Brasil)

### "No devices found"
- Certifique-se de que há dispositivos LG ThinQ na sua conta
- Verifique se os dispositivos estão online no app LG ThinQ

### "Already subscribed push" (warnings)
- Isso é normal, indica que o dispositivo já estava inscrito
- Não afeta o funcionamento do sistema

## Segurança

- **Nunca** commite o arquivo `.env` com suas credenciais
- O `.gitignore` já está configurado para ignorar este arquivo
- Mantenha seu PAT em segredo

## Referências

- [LG ThinQ Connect API (Python SDK)](https://github.com/thinq-connect/pythinqconnect)
- [Documentação oficial LG ThinQ Developer](https://connect-pat.lgthinq.com)
