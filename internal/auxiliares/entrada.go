package auxiliares

import (
	"bufio"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// LerTexto pede uma linha de texto, repetindo se vier vazio
func LerTexto(entrada *bufio.Reader, pergunta string) string {
	for {
		fmt.Print(pergunta)
		texto, _ := entrada.ReadString('\n')
		texto = strings.TrimSpace(texto)
		if texto != "" {
			return texto
		}
		fmt.Println("esse campo não pode ficar vazio, tente de novo")
	}
}

// LerTextoOpcional pede uma linha de texto, aceitando vazio (usado pra "vazio para terminar")
func LerTextoOpcional(entrada *bufio.Reader, pergunta string) string {
	fmt.Print(pergunta)
	texto, _ := entrada.ReadString('\n')
	return strings.TrimSpace(texto)
}

// LerData pede uma data DD/MM/AAAA, repetindo até ser válida.
// Devolve o texto original (não convertido) — a conversão acontece em MontarHorarios.
func LerData(entrada *bufio.Reader, pergunta string) string {
	for {
		texto := LerTexto(entrada, pergunta)
		if _, err := ConverterData(texto); err == nil {
			return texto // devolve "20/10/2000", validado mas no formato original
		}
		fmt.Println("data inválida, use o formato DD/MM/AAAA")
	}
}

// LerHora pede um horário HH:MM, repetindo até ser válido
func LerHora(entrada *bufio.Reader, pergunta string) string {
	for {
		texto := LerTexto(entrada, pergunta)
		if _, err := ConverterHora(texto); err == nil {
			return texto
		}
		fmt.Println("horário inválido, use o formato HH:MM")
	}
}

// LerPrecoCentavos pede um preço em reais, repetindo até ser válido, devolve em centavos
func LerPrecoCentavos(entrada *bufio.Reader, pergunta string) int {
	for {
		texto := LerTexto(entrada, pergunta)
		valor, err := strconv.ParseFloat(strings.Replace(texto, ",", ".", 1), 64)
		if err == nil && valor >= 0 {
			return int(math.Round(valor * 100))
		}
		fmt.Println("preço inválido, use algo como 30.50")
	}
}

// LerInteiroPositivo pede um número inteiro maior que zero, repetindo até ser válido
func LerInteiroPositivo(entrada *bufio.Reader, pergunta string) int {
	for {
		texto := LerTexto(entrada, pergunta)
		numero, err := strconv.Atoi(texto)
		if err == nil && numero > 0 {
			return numero
		}
		fmt.Println("digite um número inteiro maior que zero")
	}
}

// LerSimNao pede confirmação s/n, repetindo até ser uma das duas opções
func LerSimNao(entrada *bufio.Reader, pergunta string) bool {
	for {
		texto := strings.ToLower(LerTexto(entrada, pergunta))
		if texto == "s" {
			return true
		}
		if texto == "n" {
			return false
		}
		fmt.Println("responda com s ou n")
	}
}

func ConverterHora(entrada string) (string, error) {
	_, err := time.Parse("15:04", entrada)
	return entrada, err
}

func ConverterData(entrada string) (string, error) {
	t, err := time.Parse("02/01/2006", entrada)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}

func ConverterDataHora(entrada string) (string, error) {
	t, err := time.Parse("02/01/2006 15:04", entrada)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02T15:04"), nil
}