# oficina-mecanica-monolith

API REST de gerenciamento da oficina: ordens de serviço, orçamentos, catálogos
(clientes, veículos, serviços, insumos) e manutenção administrativa de usuários.

Go 1.26 + Fiber v3, arquitetura hexagonal, PostgreSQL. Roda no EKS.

> **Visão de arquitetura do sistema completo** (os quatro repositórios, o
> diagrama de componentes, as camadas de Terraform e o fluxo de deploy) vive em
> [`oficina-mecanica-infrastructure`](https://github.com/SOAT-15-Oficina/oficina-mecanica-infrastructure).
> Este README cobre apenas este repositório.

## O que este serviço **não** faz

| Responsabilidade | Onde vive |
|---|---|
| `POST /auth/login`, `POST /auth/register` | [`oficina-mecanica-serverless`](https://github.com/SOAT-15-Oficina/oficina-mecanica-serverless) (Lambda) |
| Painel web | [`oficina-mecanica-frontend`](https://github.com/SOAT-15-Oficina/oficina-mecanica-frontend) (S3 + CloudFront) |
| Criação de qualquer recurso AWS ou objeto Kubernetes | [`oficina-mecanica-infrastructure`](https://github.com/SOAT-15-Oficina/oficina-mecanica-infrastructure) |

Este serviço **valida** o JWT emitido pela Lambda (`internal/routes/middlewares`)
e aplica RBAC, mas nunca o emite em produção. A cópia de `AppClaims`/`ParseToken`
em `internal/auth` é intencional — os dois repositórios são independentes, e um
teste de contrato (`internal/auth/claims_test.go`) protege contra divergência.

## Estrutura

```
cmd/api/            main: modo servidor e modo `migrate`
database/           migrations goose (go:embed) + seed
docs/swagger.yaml   contrato OpenAPI (sem /auth/*)
internal/
  auth/             AppClaims + ParseToken/GenerateToken (contrato do token)
  application/      portas de persistência e notificação, erros de aplicação
  domain/           entidades e enums
  handler/          HTTP: validação, DTOs, tradução de erros
  repository/       adaptadores pgx
  routes/           registro de rotas e middlewares
  service/          casos de uso
packages/email/     provider de e-mail (mailhog | ses) + templates
tests/tools/token/  emite um JWT de desenvolvimento sem a Lambda
```

## Rodando local

```bash
cp .env.example .env      # ajuste JWT_SECRET_KEY
docker compose up --build
```

Sobe API (`:8080`), Postgres e MailHog (UI em `:8025`). As migrations rodam num
container one-shot antes da API — **não** no boot dela.

Como este compose não inclui a Lambda, não há `/auth/login`. Gere um token:

```bash
JWT_SECRET_KEY=<mesmo do .env> go run ./tests/tools/token -role admin
curl -H "Authorization: Bearer <token>" localhost:8080/customers
```

Para o fluxo completo (front + Lambda + monolito atrás de um nginx que replica o
roteamento do CloudFront), use `docker-compose.local.yml` no repositório de
infraestrutura.

## Testes

```bash
go test ./...                                # exige Postgres para os de integração
go test ./... -short                         # apenas unitários
go test ./... -coverprofile=coverage.out
```

Os testes de integração em `internal/handler/` sobem o schema real via goose e
semeiam o usuário direto na tabela `users` — o cadastro por HTTP migrou para a
Lambda, e o monolito só consome `users` para resolver quem abriu a OS.

## Qualidade

```bash
docker compose -f docker-compose.sonar.yml up -d sonarqube
go test ./... -coverprofile=coverage.out
docker compose -f docker-compose.sonar.yml run --rm sonar-scanner
```

O CI sobe uma instância equivalente como service container efêmero e o Quality
Gate **bloqueia o merge**. Como não há histórico entre execuções, o gate é
avaliado sobre o código todo, não sobre *new code*.

## Pipeline

| Gatilho | O que roda |
|---|---|
| PR → `main` | actionlint, `redocly lint`, `go test` com cobertura, SonarQube + Quality Gate |
| Push → `main` | build da imagem, push no ECR, Job de migration, `kubectl set image`, rollout, smoke check |

O deploy autentica por **OIDC** (sem access key) e resolve todos os nomes de
recurso no **SSM Parameter Store**, publicado pelo repositório de infraestrutura:

| Parâmetro | Uso |
|---|---|
| `/oficina-mecanica/prod/ecr_repository_url` | destino do `docker push` |
| `/oficina-mecanica/prod/eks_cluster_name` | `aws eks update-kubeconfig` |
| `/oficina-mecanica/prod/kube_namespace` | namespace do Deployment |
| `/oficina-mecanica/prod/api_deployment_name` | alvo do `set image` |

O `Deployment` é criado pelo Terraform com `lifecycle.ignore_changes` no campo
`image`: a **tag** é propriedade deste pipeline, o **resto do manifesto** é
propriedade do repositório de infraestrutura. Um `terraform apply` nunca reverte
um release, e um deploy daqui nunca altera probes, recursos ou HPA.

### Secrets e variables necessários

| Nome | Tipo | Conteúdo |
|---|---|---|
| `AWS_DEPLOY_ROLE_ARN` | secret | role assumida por OIDC |
| `CI_DATABASE_PASSWORD` | secret | senha do Postgres do job de teste |
| `AWS_REGION` | variable | `sa-east-1` |

## Variáveis de ambiente

Ver `.env.example`. Duas merecem atenção:

- **`JWT_SECRET_KEY`** — precisa ser byte a byte o mesmo valor da Lambda. Em
  produção vem do Secrets Manager via `kubernetes_secret`.
- **`APP_BASE_URL`** — usada para montar os links de aprovação enviados ao
  cliente por e-mail (`internal/service/budget_service.go`). Em produção é o
  domínio do CloudFront seguido de `/api`.

## E-mail

`EMAIL_PROVIDER=mailhog` local, `EMAIL_PROVIDER=ses` em produção. O provider SES
usa a API v2 e resolve credenciais pela cadeia padrão do SDK — no cluster isso
cai no **IRSA** do ServiceAccount, então não existe access key em lugar nenhum.

O SES opera em **sandbox**: só entrega para endereços verificados, com teto de
200 e-mails/24h e 1/segundo. Endereço não verificado retorna erro, e o provider
propaga esse erro em vez de engolir.
