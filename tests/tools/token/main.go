// Ferramenta de desenvolvimento: emite um JWT valido para este servico sem
// depender da Lambda do oficina-mecanica-serverless.
//
//	go run ./tests/tools/token -user mecanico -role employee
//
// O token so e aceito se JWT_SECRET_KEY for o mesmo configurado na API.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/auth"
)

func main() {
	user := flag.String("user", "dev", "username embutido no claim")
	role := flag.String("role", "admin", "admin ou employee")
	secret := flag.String("secret", os.Getenv("JWT_SECRET_KEY"), "chave de assinatura")
	flag.Parse()

	if *secret == "" {
		fmt.Fprintln(os.Stderr, "defina JWT_SECRET_KEY ou passe -secret")
		os.Exit(1)
	}

	token, err := auth.GenerateToken(*user, *role, *secret)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(token)
}
