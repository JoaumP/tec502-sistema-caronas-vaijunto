package auxiliares

import "fmt"

func ImprimirTitulo(titulo string) {
	linha := "══════════════════════════════════════"
	fmt.Println("\n" + linha)
	fmt.Println("  " + titulo)
	fmt.Println(linha)
}

func ImprimirMenu(titulo string, opcoes []string) {
	ImprimirTitulo(titulo)
	for i, o := range opcoes {
		fmt.Printf("  %d. %s\n", i+1, o)
	}
	fmt.Print("\n> ")
}

func ImprimirSeparador() {
	fmt.Println("────────────────────────────────────────")
}