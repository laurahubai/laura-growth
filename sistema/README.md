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
cd backend && set -a && . ../.env && set +a && go test -cover ./...
pnpm typecheck
pnpm e2e
```

Para `pnpm e2e`, os pré-requisitos são:

1. **Banco exclusivo de E2E** cujo nome termine com `_teste` ou `_e2e` (ex.: `laura_growth_e2e`).
2. **Variáveis no `.env`** (variáveis do ambiente sobrescrevem o `.env`):
   - `LG_E2E_BANCO_URL`: obrigatória (os testes abortam antes de iniciar se estiver ausente ou se o nome do banco não terminar com `_teste` ou `_e2e`).
   - `LG_CHAVE_CIFRAGEM`, `LG_SEMENTE_NOME`, `LG_SEMENTE_EMAIL`, `LG_SEMENTE_SENHA`.
   - Opcionais: `LG_E2E_API_ENDERECO` (padrão `127.0.0.1:18090`; porta `8090` recusada) e `LG_E2E_WEB_PORTA` (padrão `13200`; porta `3200` recusada). A compilação do E2E usa `.next-e2e` para não sobrescrever `.next`.
3. **Chromium do Playwright instalado** (na primeira execução):

```bash
pnpm --filter @laura-growth/web exec playwright install chromium
```

Os testes criam um usuário comum temporário com senha aleatória por execução e o removem ao final.

## Administração e IA (Etapa 2A-1)

- **`LG_CHAVE_CIFRAGEM` é obrigatória**: o servidor Go não sobe sem ela (mínimo 32 caracteres). Gere com `openssl rand -base64 48`, coloque apenas em `.env` e **nunca versione** `.env`, chaves ou senhas.
- **Fornecedor simulado e modelo com falha**: o tipo `simulado` responde localmente sem internet (`simulado-texto` e `simulado-imagem`). Qualquer modelo cujo identificador contenha `falha` (ex.: `simulado-falha`) falha de propósito para validar a troca automática para o modelo reserva e o registro de tentativas.
- **Geração real de imagens (OpenAI)**: o fluxo completo de imagem foi validado nos testes com o conector simulado; a geração real na OpenAI ainda deve ser confirmada com uma chave contratada válida.
