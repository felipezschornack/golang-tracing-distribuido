# Tracing Distribuido - OTEL(Open Telemetry) e Zipkin
A implementação está dividida em 2 microsserviços:

## Microsserviço A (responsável pelo input):

O microsserviço recebe um input de 8 dígitos via POST, através do payload: 
```
{ 
    "cep": "29902555" 
}
```

Verifica se o input é valido (contem 8 dígitos) e é uma STRING. 

Caso seja válido, encaminha a requisição para o microsserviço B via HTTP.

Caso não seja válido, retorna:

```
Código HTTP: 422
Mensagem: invalid zipcode
```

## Serviço B (responsável pela orquestração):
Através de uma requisição HTTP GET, o microsserviço recebe um CEP válido de 8 digitos. Com base nele, realiza a pesquisa do CEP, encontra o nome da localização e, a partir disso, busca e retorna as temperaturas correspondente à localização em: Celsius, Fahrenheit, Kelvin juntamente com o nome da localização.

Em caso de sucesso:
```
Código HTTP: 200
Response Body: 
{ 
    "city: "São Paulo", 
    "temp_C": 28.5, 
    "temp_F": 28.5, 
    "temp_K": 28.5 
}
```

Em caso de falha, caso o CEP não seja válido (com formato correto):
```
Código HTTP: 422
Mensagem: invalid zipcode
```

​​​Em caso de falha, caso o CEP não seja encontrado:
```
Código HTTP: 404
Mensagem: can not find zipcode
```

## Tracing distribuido

Os microsserviços implementam tracing distribuído.

Utilizou-se span para medir o tempo de resposta do serviço de busca de CEP e busca de temperatura

# Executando a aplicação utilizando Docker
Com o Docker instalado em sua estação de trabalho (https://www.docker.com/), execute o comando:
```
docker compose up -d
```

# Acessando a aplicação Localmente
Depois de instalar e executar a aplicação via Docker, a aplicação estará disponível para uso em http://localhost:8080/weather.

Executar o seguinte comando:

```
curl --location 'http://localhost:8080/weather' \
--header 'Content-Type: application/json' \
--data '{
    "cep": "15480-001"
}'
```

O resultado esperado é algo como:

```
{
    "city": "Orindiúva",
    "temp_C": 23.5,
    "temp_F": 74.3,
    "temp_K": 296.5
}
```

# Verificando o trace gerado

## Zipkin
http://localhost:9411

## Jaeger
http://localhost:16686

## Prometheus
http://localhost:9090

## Grafana
http://localhost:3001

- Acesso com username 'admin' e password 'admin'
- Datasource do Prometheus: http://prometheus:9090
- Dashboard para Golang: https://grafana.com/grafana/dashboards/10826-go-metrics/