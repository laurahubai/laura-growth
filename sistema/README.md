# Plataforma Laura Growth — código

Código de verdade da plataforma (o protótipo com dados de exemplo continua em `../plataforma`).
Documentação e decisões: cofre `../obsidian/01 - Projetos/Plataforma Laura Growth/`.

| Pasta | O que é |
|---|---|
| `apps/web` | Telas (Next.js). Chama o servidor pelo mesmo endereço, em `/api/...` |
| `backend` | Servidor (Go) com a API, o login e o banco |
| `packages/contratos` | Formato dos dados trocados entre telas e servidor |
| `deploy` | Banco PostgreSQL de desenvolvimento (Docker Compose) |

## Rodar em desenvolvimento (máquina `gcp-lhac-compozyos-001`)

```bash
cp .env.example .env            # preencher a semente (e-mail e senha de desenvolvimento)
docker compose -f deploy/compose.yaml --env-file .env up -d
cd backend && set -a && . ../.env && set +a && go run . migrar && go run . semear && go run . servir
pnpm install && pnpm dev        # telas em http://localhost:3200
```

Portas: telas `3200`, API `8090`, banco `55433` (todas só em `127.0.0.1`). Acesso do Mac por túnel SSH.

## Stage

https://stage-growth.laurahub.com.br — só para os IPs liberados no nginx de stage (mesma lista do painel da Laura Hub).
Roda como serviço: `laura-growth-api` (Go, 127.0.0.1:8090) e `laura-growth-web` (Next, 172.20.0.1:3210). Para publicar uma mudança:

```bash
cd backend && go build -o laura-growth-api . && set -a && . ../.env && set +a && ./laura-growth-api migrar
cd ../apps/web && NODE_ENV=production ./node_modules/.bin/next build
sudo systemctl restart laura-growth-api laura-growth-web
```

## Testes

```bash
cd backend && set -a && . ../.env && set +a && go test ./...
pnpm typecheck
```
