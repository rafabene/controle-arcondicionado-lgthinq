# Economizador de Energia - LG ThinQ

Aplicação Go que monitora e controla automaticamente a temperatura de ar condicionados LG através da API ThinQ, garantindo que a temperatura nunca fique abaixo do mínimo configurado para economia de energia.

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
│       ├── mqtt_models.go   # Modelos de dados do MQTT
│       ├── state.go         # Estado do dispositivo e parsing de power/temperatura
│       └── state_test.go    # Testes do parsing de estado
├── .env                     # Variáveis de ambiente (não versionado)
├── .env.example             # Exemplo de configuração (versionado)
├── .gitignore
├── go.mod
├── go.sum
├── Makefile                 # Comandos de build e execução
└── README.md
```

## Funcionalidades

- Conexão em tempo real via MQTT com dispositivos LG ThinQ
- Detecção automática de mudanças de temperatura
- Ajuste automático quando a temperatura fica abaixo do mínimo configurado
- Ajuste aplicado apenas com o aparelho ligado (comandos de temperatura são rejeitados pela API em `POWER OFF`)
- Verificação do estado de todos os dispositivos na inicialização, corrigindo aparelhos que já estão ligados abaixo do mínimo
- Temperatura mínima configurável via variável de ambiente
- Suporte para múltiplos dispositivos simultaneamente

## Requisitos

- Go 1.26 ou superior
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
MIN_TEMPERATURE=23
```

**Notas:**
- O Client ID é gerado automaticamente pela aplicação, não precisa configurar
- `MIN_TEMPERATURE` define a temperatura mínima permitida (recomendado: 23°C; piso aceito: 21°C)

## Uso

Para compilar e executar:

```bash
make build   # Compila o binário
make run     # Compila e executa
make clean   # Remove o binário
```

Ou diretamente com Go:

```bash
go run cmd/economizador/main.go
```

O sistema irá:
1. Conectar aos servidores LG ThinQ
2. Listar todos os dispositivos encontrados
3. Inscrever-se para receber eventos de cada dispositivo
4. Monitorar mudanças de temperatura em tempo real
5. Ajustar automaticamente para a temperatura mínima quando detectar valor abaixo do configurado

Para parar o programa, pressione `Ctrl+C`.

## Como Funciona

1. **Autenticação**: Usa o Personal Access Token para autenticar com a API LG ThinQ
2. **Descoberta**: Lista todos os dispositivos de ar condicionado na conta
3. **Registro MQTT**: Registra o cliente e obtém certificados para conexão MQTT segura (mTLS)
4. **Inscrição**: Inscreve-se para receber eventos de cada dispositivo
5. **Verificação inicial**: Consulta o estado de cada dispositivo e ajusta os que já estão ligados abaixo do mínimo
6. **Monitoramento**: Escuta eventos MQTT em tempo real
7. **Controle**: Quando detecta `targetTemperature` abaixo do mínimo configurado em um aparelho ligado, envia comando para ajustar

## Configuração

### Temperatura Mínima

Configure a temperatura mínima permitida no arquivo `.env`:

```env
MIN_TEMPERATURE=23
```

- O valor recomendado é **23 °C** (veja [Por que 23 °C?](#por-que-23-c))
- Valores abaixo de 21 serão automaticamente ajustados para 21
- Quando o ar condicionado é configurado para uma temperatura abaixo do mínimo, o sistema ajusta automaticamente
- Temperaturas iguais ou acima do mínimo são permitidas

### Por que 23 °C?

O código aceita 21 °C como piso, mas o valor usado na operação é 23 °C porque é
onde conforto, saúde e consumo de energia se encontram:

- **É o limite inferior da faixa recomendada pela ANVISA.** A Resolução RE nº 09/2003,
  do Ministério da Saúde, recomenda de 23 °C a 26 °C para ambientes climatizados de
  uso coletivo no verão. Abaixo de 23 °C o ambiente sai da faixa considerada adequada.
- **É a recomendação do Inmetro para o termostato.** O Inmetro orienta ajustar o
  aparelho para 23 °C por ser a faixa que equilibra conforto térmico e eficiência
  energética, e lembra que temperaturas extremas (como 17 °C) não resfriam o ambiente
  mais rápido: apenas mantêm o compressor ligado por mais tempo.
- **Cada grau a menos custa energia.** Segundo o Procel, o termostato deve ficar entre
  23 °C e 24 °C, e cada grau abaixo disso pode elevar o consumo em até 8%. Sair de
  21 °C para 23 °C representa, na ordem de 16% menos energia no compressor.
- **23 °C ainda desumidifica.** Quanto mais alto o setpoint, menos frio fica o
  evaporador e menos água ele condensa, deixando o ar mais úmido. Em 23 °C o aparelho
  ainda condensa o suficiente para o conforto de um dia quente, sem o frio
  desconfortável de 21 °C.
- **21 °C é piso técnico, não recomendação.** O piso existe para rejeitar configurações
  erradas; operar no piso é o cenário mais caro possível, e não o objetivo do projeto.

## API Endpoints Utilizados

- `GET /devices` - Lista dispositivos
- `GET /devices/{deviceId}/state` - Obtém o estado atual do dispositivo
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

### "Command not supported in POWER OFF" (código 2304)
- Ocorria quando o ajuste era enviado com o aparelho desligado
- O programa agora verifica o estado de energia (`operation.airConOperationMode`) antes de enviar o comando e ignora aparelhos em `POWER_OFF`

## Segurança

- **Nunca** commite o arquivo `.env` com suas credenciais
- O `.gitignore` já está configurado para ignorar este arquivo
- Mantenha seu PAT em segredo

## Referências

- [LG ThinQ Connect API (Python SDK)](https://github.com/thinq-connect/pythinqconnect)
- [Documentação oficial LG ThinQ Developer](https://connect-pat.lgthinq.com)
