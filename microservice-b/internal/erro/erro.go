package erro

type Erro struct {
	Status  int
	Message string
}

func New(status int, message string) *Erro {
	return &Erro{Status: status, Message: message}
}
