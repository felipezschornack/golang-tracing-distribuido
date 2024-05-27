package viacep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/felipezschornack/golang-tracing-distribuido/service-b/internal/erro"
	"go.opentelemetry.io/otel/trace"
)

// For more information, please see https://viacep.com.br
type ViaCEP struct {
	Cep         string `json:"cep,omitempty"`
	Logradouro  string `json:"logradouro,omitempty"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro,omitempty"`
	Localidade  string `json:"localidade"`
	Uf          string `json:"uf,omitempty"`
	Ibge        string `json:"ibge,omitempty"`
	Gia         string `json:"gia,omitempty"`
	Ddd         string `json:"ddd,omitempty"`
	Siafi       string `json:"siafi,omitempty"`
}

type ErroViaCEP struct {
	Erro string `json:"erro"`
}

const INVALID_ZIPCODE_MSG = "invalid zipcode"
const CANNOT_FIND_ZIPCODE_MSG = "can not find zipcode"

func BuscaCep(zipcode string, ctx context.Context, tracer trace.Tracer) (*ViaCEP, *erro.Erro) {
	if zipcode == "" {
		return nil, erro.New(http.StatusBadRequest, INVALID_ZIPCODE_MSG)
	}

	ctx, span := tracer.Start(ctx, "Span_ViaCepAPI_Request")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://viacep.com.br/ws/%s/json/", url.QueryEscape(zipcode)), nil)
	if err != nil {
		return nil, erro.New(http.StatusInternalServerError, err.Error())
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, erro.New(http.StatusBadRequest, err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			return nil, erro.New(http.StatusInternalServerError, err.Error())
		}

		var data ViaCEP
		err = json.Unmarshal(body, &data)
		if err == nil {
			if data.Localidade != "" {
				return &data, nil
			} else {
				return nil, erro.New(http.StatusNotFound, CANNOT_FIND_ZIPCODE_MSG)
			}
		} else {
			var erroViaCep ErroViaCEP // se for codigo de erro da API (pode ser 200 e erro == true)
			err = json.Unmarshal(body, &erroViaCep)
			if err != nil {
				return nil, erro.New(http.StatusInternalServerError, err.Error())
			} else {
				return nil, erro.New(http.StatusUnprocessableEntity, INVALID_ZIPCODE_MSG)
			}
		}
	} else if resp.StatusCode == 404 {
		return nil, erro.New(http.StatusNotFound, CANNOT_FIND_ZIPCODE_MSG)
	} else {
		return nil, erro.New(http.StatusBadRequest, "erro ao fazer requisicao")
	}
}
