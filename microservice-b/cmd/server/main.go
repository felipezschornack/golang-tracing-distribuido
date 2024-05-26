/*
Requisitos - Serviço B (responsável pela orquestração):

# O sistema deve receber um CEP válido de 8 digitos

O sistema deve realizar a pesquisa do CEP e encontrar o nome da localização, a partir disso, deverá retornar as temperaturas e formata-lás em:

	Celsius, Fahrenheit, Kelvin juntamente com o nome da localização.

O sistema deve responder adequadamente nos seguintes cenários:

Em caso de sucesso:

	Código HTTP: 200
	Response Body: { "city: "São Paulo", "temp_C": 28.5, "temp_F": 28.5, "temp_K": 28.5 }

Em caso de falha, caso o CEP não seja válido (com formato correto):

	Código HTTP: 422
	Mensagem: invalid zipcode

​​​Em caso de falha, caso o CEP não seja encontrado:

	Código HTTP: 404
	Mensagem: can not find zipcode
*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/felipezschornack/golang-tracing-distribuido/service-b/internal/backend/remote/viacep"
	"github.com/felipezschornack/golang-tracing-distribuido/service-b/internal/backend/remote/weatherapi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CepRequestDTO struct {
	Cep string `json:"cep"`
}

type Error struct {
	Message string `json:"message"`
}

var apiKey string
var tracer trace.Tracer

func main() {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := initProvider("microservice-b", "otel-collector:4317")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()
	tracer = otel.Tracer("otel-tracer")

	apiKey = os.Getenv("WEATHER_API_KEY")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(logRequest)
	r.Handle("/metrics", promhttp.Handler())
	r.Route("/weather", func(r chi.Router) {
		r.Get("/{zipcode}", weatherHandler)
	})
	http.ListenAndServe(":8081", r)

	select {
	case <-sigCh:
		log.Println("Shutting down gracefully, CTRL+C pressed...")
	case <-ctx.Done():
		log.Println("Shutting down due to other reason...")
	}

	// Create a timeout context for the graceful shutdown
	_, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	//ctx, span := tracer.Start(ctx, "Span_Service_B_Request")
	//defer span.End()

	zipcode := chi.URLParam(r, "zipcode")
	if zipcode == "" {
		w.WriteHeader(http.StatusBadRequest)
		error := Error{Message: "Parameter 'cep' is mandatory"}
		json.NewEncoder(w).Encode(error)
		return
	}

	// chamar aqui viacep
	data, err := viacep.BuscaCep(zipcode, ctx, tracer)

	if err != nil {
		w.WriteHeader(err.Status)
		json.NewEncoder(w).Encode(err.Message)
		return
	}

	// chamar aqui a weatherAPI
	weatherData, err := weatherapi.GetWeather(data.Localidade, apiKey, ctx, tracer)

	if err != nil {
		w.WriteHeader(err.Status)
		json.NewEncoder(w).Encode(err.Message)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(weatherData)
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func initProvider(serviceName, collectorURL string) (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, collectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider.Shutdown, nil
}
