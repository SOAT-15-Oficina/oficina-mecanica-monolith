package auth

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Contrato de token entre oficina-mecanica-serverless (emite) e este servico
// (valida). Os dois repositorios sao independentes e cada um tem sua copia de
// AppClaims/ParseToken -- sao ~40 linhas, nao valem um modulo compartilhado nem
// um quinto repositorio.
//
// O que impede as copias de divergirem e este fixture: `testdata/token.golden`
// e byte a byte o mesmo arquivo nos dois repos, emitido pelo gerador que vive
// no -serverless (`go run ./tools/gentoken`). Se alguem renomear um claim,
// trocar o algoritmo de assinatura ou mexer no formato, o build quebra aqui em
// vez de virar 401 em producao.
//
// Ao atualizar o fixture: regenere no -serverless e copie o arquivo para os
// dois repositorios no mesmo PR.
const (
	contractSecret   = "contract-test-secret"
	contractUser     = "contract-user"
	contractRole     = "admin"
	contractGoldenAt = "testdata/token.golden"
)

func readGoldenToken(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(contractGoldenAt)
	require.NoError(t, err, "fixture de contrato ausente; copie de oficina-mecanica-serverless")

	return strings.TrimSpace(string(raw))
}

func TestContract_ParsesTokenIssuedByServerless(t *testing.T) {
	claims, err := ParseToken(readGoldenToken(t), contractSecret)

	require.NoError(t, err, "o token emitido pela Lambda deixou de ser aceito aqui")
	assert.Equal(t, contractUser, claims.User, `claim "user" divergiu entre os repos`)
	assert.Equal(t, contractRole, claims.Role, `claim "role" divergiu entre os repos`)
}

// O fixture tem expiracao em 2099. Se este teste falhar por token expirado,
// algo mudou no formato -- nao e o relogio.
func TestContract_GoldenTokenIsNotExpired(t *testing.T) {
	_, err := ParseToken(readGoldenToken(t), contractSecret)

	require.NoError(t, err)
}

func TestContract_GoldenTokenIsHS256(t *testing.T) {
	parts := strings.Split(readGoldenToken(t), ".")
	require.Len(t, parts, 3, "token golden malformado")

	// O header e o mesmo que jwt.SigningMethodHS256 produz. Trocar para RS256
	// no emissor exigiria mudar a validacao aqui tambem.
	assert.Equal(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", parts[0])
}
