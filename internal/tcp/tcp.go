package tcp

import "bufio"

// EnviarMensagem escreve os dados no socket, seguidos de uma quebra de linha
func EnviarMensagem(w *bufio.Writer, dados []byte) error {
	if _, err := w.Write(dados); err != nil {
		return err
	}
	if err := w.WriteByte('\n'); err != nil {
		return err
	}
	return w.Flush()
}

// LerMensagem lê do socket até encontrar uma quebra de linha
func LerMensagem(r *bufio.Reader) ([]byte, error) {
	linha, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	// remove o \n do final antes de devolver
	return linha[:len(linha)-1], nil
}