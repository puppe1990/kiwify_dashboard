package kiwify

import (
	"errors"
	"fmt"
)

// APIError is a typed Kiwify Public API error with a PT-BR user-facing message.
type APIError struct {
	Status      int
	Code        string
	Message     string
	UserMessage string
	Body        string
}

func (e *APIError) Error() string {
	if e == nil {
		return "kiwify: <nil>"
	}
	if e.Message != "" {
		return fmt.Sprintf("kiwify API %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("kiwify API %d", e.Status)
}

// AsAPIError unwraps err to *APIError when possible.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

func userMessageForStatus(status int, apiMessage string) string {
	switch {
	case status == 401 || status == 403:
		return "Credenciais Kiwify inválidas ou token expirado. Verifique Configurações."
	case status == 429:
		return "Limite da API Kiwify atingido (100/min). Aguarde um minuto e tente de novo."
	case status == 400:
		if apiMessage != "" {
			return apiMessage
		}
		return "Requisição inválida."
	case status == 404:
		return "Recurso não encontrado na API Kiwify (404). Verifique se a conta e as permissões da API Key estão corretas."
	case status >= 500 && status <= 599:
		return "Erro no servidor da Kiwify. Tente mais tarde."
	default:
		if apiMessage != "" {
			return apiMessage
		}
		return fmt.Sprintf("Erro da API Kiwify (%d).", status)
	}
}
