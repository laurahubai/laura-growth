---
tipo: projeto
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-25
---
# Especificação — Etapa 2A-1 (Cadastro de IA por chave de API)

Spec de construção da etapa 2A-1 do [[Roteiro de Construção - Fase 1A]], a partir do [[prd-lauragrowth20|PRD 2.0]] v1.4 (§3, §8.3, §9, §11, §13, §15.1) e da [[Revisão PRD 2.0 - Etapa 2A]]. Não rediscute decisões; o que é escolha de implementação está marcado **(a confirmar)**.

Branch: `etapa-2a-1-cadastro-ia`, a partir da `main` (f5d3587). Código em `sistema/`.

## Objetivo
O administrador da plataforma (só o Paulo) cadastra fornecedores de IA por chave de API, modelos, preços e limites, escolhe principal e reserva de cada atividade e testa; toda chamada grava tokens e US$.

## Critérios de aceite (cada um com teste automático)
| AC | O que o teste prova |
|---|---|
| AC29 | Testar (modelo ou atividade) grava registro com tokens, preço usado e US$; simulado marcado como simulado. Valor real: aguarda chave contratada |
| AC30 | Pessoa de organização sem a marca de administrador recebe 403 em todas as rotas `/api/admin/*`; a tela de Operação não mostra fornecedores. Atividade de organização de fora da casa nunca usa fornecedor com uso "só a casa" |
| AC32 | Depois de salva, a chave não volta em nenhuma resposta (só os 4 últimos); não aparece em logs; fica cifrada no banco |
| AC33 | Principal falha → reserva conclui; o registro mostra qual atendeu e a falha do principal |
| AC34 | Gasto do mês ≥ percentual → aviso na Administração; ≥ limite → vai para a reserva; reserva também no limite → para e avisa. Limite do modelo vale no lugar do fornecedor |
| AC35 | Testar texto ("Oi, qual modelo é você?") e imagem ("um círculo azul sobre fundo branco") mostram resultado, tokens e US$; simulado marcado |
| AC36 | Atividade sem principal ou sem reserva não roda e diz o que falta |
| AC37 | Quem não é administrador não vê o menu Administração e o servidor recusa (403) |
| AC38 | "Ler modelos disponíveis" traz a lista; chave inválida ou fornecedor fora do ar → mensagem clara, sem a chave |
| AC39 | Endereço sem `https` ou que aponte para rede interna é recusado no cadastro e também na hora da conexão |

## Dados (migração `0003_cadastro_ia.sql`, só acrescenta)
- `usuarios.admin_plataforma` (padrão falso; a semente marca a pessoa da semente).
- `organizacoes.da_casa` (padrão falso; a semente marca Laura Growth e Laura Hub).
- `ia_fornecedores`: nome, tipo (`openai`, `compativel_openai`, `anthropic`, `simulado`), cadastro (`chave_api`; `assinatura` reservado à 2A-2), endereço, chave cifrada, 4 últimos, limite mensal US$ (obrigatório, > 0), aviso % (padrão 80), uso permitido (`todas` ou `casa`), ativo.
- `ia_modelos`: fornecedor, identificador, serviço (`texto`/`imagem`), origem (`lido`/`manual`), limite e aviso próprios (opcionais), ativo.
- `ia_precos`: modelo, unidade (`token_entrada`, `token_saida`, `imagem`), preço US$, vigência a partir de. Só acrescenta (histórico preservado).
- `ia_rotas`: atividade (`chat_cmo`, `plano`, `pauta`, `texto`, `imagem`, `revisao`), principal, reserva.
- `ia_execucoes` (AgentRun): correlação, atividade (ou `teste`), organização (opcional), pessoa, fornecedor e modelo (com nome copiado), tentativa, atendido pela reserva, status, erro (sem segredo), tokens de entrada e saída, imagens, preços usados, US$, estimado (falso na 2A-1), simulado.
- Nada é apagado: fornecedor e modelo são **desativados**, não removidos.

## Regras
- **Chave:** AES-256-GCM, segredo derivado (HKDF-SHA256) de `LG_CHAVE_CIFRAGEM`. O servidor não sobe sem ela. Chave nunca em log, resposta ou mensagem de erro (mensagens do fornecedor são traduzidas e limpas).
- **Endereço:** OpenAI `https://api.openai.com/v1`, Anthropic `https://api.anthropic.com/v1`, compatível com OpenAI obrigatório e livre, simulado sem endereço. Só `https`; recusa endereços internos (loopback, privados, link-local/metadados, CGNAT, IPv6 locais) no cadastro e na conexão; sem seguir redirecionamentos; tempo máximo por chamada.
- **Preço:** tokens em **US$ por 1 milhão de tokens**; imagem em US$ por imagem **(a confirmar)**. Testar e atividades exigem preço vigente; sem preço, recusa com mensagem clara **(a confirmar)**.
- **Mês do limite:** mês do calendário no fuso de São Paulo **(a confirmar)**. Gasto = soma dos registros do mês.
- **Testar respeita o limite** (no limite, recusa) **(a confirmar)**. Não exige reserva.
- **Simulado:** responde sem internet, de forma previsível; modelos `simulado-texto` e `simulado-imagem`; um modelo cujo nome contém `falha` sempre falha, para ver a reserva assumir **(a confirmar)**.
- **Testar atividade:** botão na tela de Atividades que roda a atividade com principal e reserva, para ver AC33, AC34 e AC36 na prática **(a confirmar)**.
- Imagem do Anthropic não é aceita (o fornecedor não gera imagem).

## Rotas (servidor Go; todas exigem sessão e marca de administrador)
`GET /api/sessao` passa a trazer `usuario.adminPlataforma`.

| Rota | Faz |
|---|---|
| `GET/POST /api/admin/fornecedores` · `PATCH /api/admin/fornecedores/{id}` | listar (com gasto do mês e situação: normal, aviso, limite), cadastrar, editar, trocar chave, desativar |
| `POST /api/admin/fornecedores/{id}/ler-modelos` | lê a lista no fornecedor |
| `GET/POST /api/admin/fornecedores/{id}/modelos` · `PATCH /api/admin/modelos/{id}` | modelos lidos ou à mão; limite próprio; desativar |
| `GET/POST /api/admin/modelos/{id}/precos` | preços com vigência |
| `POST /api/admin/modelos/{id}/testar` | teste por serviço |
| `GET /api/admin/atividades` · `PUT /api/admin/atividades/{atividade}` · `POST .../testar` | principal e reserva; teste da atividade |
| `GET /api/admin/execucoes` · `GET /api/admin/avisos` | registros recentes; avisos de limite |

Contrato detalhado (campos JSON): `sistema/packages/contratos` e comentários no código.

## Telas
- **Administração** em `/admin`, com menu e cor de cabeçalho próprios; aparece no menu da Operação só para administrador; quem não é vai para `/inicio`.
- Fornecedores (lista, situação do limite, novo/editar com endereço preenchido pelo tipo, chave mostrada como `••••abcd`).
- Fornecedor → modelos ("Ler modelos disponíveis", incluir à mão, preços com vigência, limite próprio, **Testar** com resultado, tokens, US$ e selo **Simulado**).
- Atividades (principal e reserva por atividade, "o que falta", Testar atividade).
- Registros (execuções com tokens, US$, reserva, simulado).

## Testes obrigatórios (PRD §15.1)
Um teste por AC acima; testes antigos passando; `go vet` e typecheck limpos; cobertura do servidor ≥ 60%; Playwright no caminho principal (entrar → Administração → cadastrar simulado → ler modelos → preço → testar → atividade com reserva); migração só acrescenta, com o desfazer em comentário; `pg_dump` antes de migrar o banco da VM.

## Fatias
| # | Fatia | Depende de |
|---|---|---|
| S1 | Base do servidor: migração, semente, marca de administrador (AC37/AC30), cifragem (AC32), proteção de endereço (AC39), cadastro de fornecedores | — |
| S2 | Conectores (OpenAI, compatível, Anthropic, simulado), ler modelos (AC38), modelos e preços | S1 |
| S3 | Atividades e executor: principal e reserva (AC36, AC33), limites e avisos (AC34), registros e Testar (AC29, AC35) | S2 |
| S4 | Telas da Administração e contratos | S1 (contrato desta spec); em paralelo com S2/S3 |
| S5 | Playwright, cobertura, README e ajustes finais | S3, S4 |

## Fica de fora
Assinatura logada (2A-2); teste com chave real (aguarda fornecedor contratado); alçadas por pessoa (2B); painel de custos da Operação (etapa 7); cofre de senhas dedicado.

## Relacionado
[[Roteiro de Construção - Fase 1A]] · [[prd-lauragrowth20|PRD LauraGrowth 2.0]] · [[Revisão PRD 2.0 - Etapa 2A]] · [[Tecnologias de Base]] · [[Índice - Projetos]] · [[Home]]
