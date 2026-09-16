package auxiliares

import "time"

// MontarHorarios monta os horarios de saida/chegada de um trecho.
// dataInterna já deve estar no formato "2006-01-02" (use ConverterData antes, se vier do usuário).
// Se a hora de chegada for menor que a de saida, assume que a chegada é no dia seguinte.
func MontarHorarios(dataInterna, horaSaida, horaChegada string) (string, string, error) {
	saida := dataInterna + "T" + horaSaida
	chegada := dataInterna + "T" + horaChegada

	if _, err := time.Parse("2006-01-02T15:04", saida); err != nil {
		return "", "", err
	}

	if chegada < saida {
		t, err := time.Parse("2006-01-02T15:04", chegada)
		if err != nil {
			return "", "", err
		}
		chegada = t.AddDate(0, 0, 1).Format("2006-01-02T15:04")
	}

	return saida, chegada, nil
}

// FormatarHorario converte "2026-09-15T12:00" para "15/09 12:00"
func FormatarHorario(h string) string {
	t, err := time.Parse("2006-01-02T15:04", h)
	if err != nil {
		return h
	}
	return t.Format("02/01 15:04")
}