package auxiliares

import (
	"time"
)

// MontarHorarios monta os horarios de saida/chegada de um trecho.
// Se a hora de chegada for menor que a de saida, assume que a chegada
// e no dia seguinte (viagem que passa da meia-noite).
func MontarHorarios(data, horaSaida, horaChegada string) (string, string, error) {
	saida, err := ConverterDataHora(data + " " + horaSaida)
	if err != nil {
		return "", "", err
	}

	chegada, err := ConverterDataHora(data + " " + horaChegada)
	if err != nil {
		return "", "", err
	}

	if chegada < saida { // string compara certo nesse formato (AAAA-MM-DDTHH:MM)
		t, _ := time.Parse("2006-01-02T15:04", chegada)
		chegada = t.AddDate(0, 0, 1).Format("2006-01-02T15:04")
	}

	return saida, chegada, nil
}