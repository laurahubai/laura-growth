---
tipo: projeto
status: em validação
area: Produto e Agentes de IA
criado: 2026-11-26
---
# Relatório — Etapa 2B (Acessos por pessoa e por alçada)

Resumo da construção da etapa 2B do [[Roteiro de Construção - Fase 1A]], conforme a [[Especificação - Etapa 2B]]. Branch: `etapa-2b-acessos` (feita a partir da `etapa-2a-1-cadastro-ia`, porque a 2A-1 ainda não está na `main`).

## O que foi feito
- **Alçadas por pessoa:** as 8 alçadas aprovadas podem ser marcadas uma a uma para cada pessoa da organização. A mudança vale já no pedido seguinte da pessoa. O dono da organização tem todas as alçadas sempre e ninguém pode alterá-lo nem removê-lo. Ninguém altera as próprias alçadas.
- **Conjuntos prontos:** Somente leitura, Criador de conteúdo, Aprovador e Gestor de marketing. Escolher um conjunto só pré-marca as caixas, que podem ser ajustadas antes de salvar.
- **Convites por e-mail:** quem tem a alçada "Convidar pessoas e dar acessos" convida um e-mail já escolhendo as alçadas. O link vale uma vez e por 7 dias. A pessoa cria a conta (nome e senha) ou, se já tem conta, entra com o mesmo e-mail e aceita. É possível reenviar (o link antigo deixa de valer), cancelar e copiar o link logo após criar ou reenviar.
- **Proteção no servidor:** sem a alçada, a ação é recusada pelo servidor, mesmo forçando pelo endereço da página. Dados de outra organização não aparecem. Hoje isso já protege o kit de marca: ver o kit exige "Ver a marca e os posts" e editar exige "Conversar com o CMO e editar o DNA da marca". As demais alçadas já estão prontas para as próximas etapas.
- **Histórico:** todo convite, aceite, cancelamento, concessão, retirada e remoção fica registrado (quem, para quem, o quê e quando) e aparece na tela Pessoas. O registro não pode ser alterado nem apagado.
- **Limites contra abuso:** até 30 convites pendentes e 20 envios por hora por organização, com mensagem clara.
- **Telas:** Operação → **Pessoas**; página pública do convite; menus e botões escondidos quando falta a alçada; aviso "sem acesso" quando o servidor recusa; Administração → **Equipe** (dar ou retirar acesso de administrador da plataforma para pessoas da Laura Hub e da Laura Growth; o último administrador não pode ser removido).

## Testes rodados (26/11/2026, na branch integrada)
- Testes do servidor (Go): `go vet` sem avisos, formatação ok, `go test` **aprovado**, cobertura de 72,7%.
- Verificação de tipos e montagem das telas: **aprovadas**.
- Testes de ponta a ponta no navegador (Playwright): **8 de 8 aprovados**, incluindo o fluxo convite → aceite → pessoa limitada bloqueada pelo endereço, o fluxo do dono, os limites, o convite vencido, a tela Equipe e os testes da etapa 2A-1.
- Cada parte (S1, S2, S3) passou por duas revisões (geral e segurança). Nenhum problema bloqueante ficou aberto; os ajustes importantes foram feitos.

## O que ficou de fora e por quê
- **Envio real do e-mail:** falta a chave do Zoho (`LG_EMAIL_ZOHO_CHAVE`) e o endereço público (`LG_ENDERECO_PUBLICO`) no `.env`. Até lá, o convite fica como "não enviado" e o link pode ser copiado na tela. Os testes usam um carteiro simulado.
- **Implantação no stage:** não foi feita. É preciso fazer cópia do banco, aplicar a migração 0004 e trocar o programa do servidor. Isso fica para quando você quiser testar, porque o stage usa o mesmo banco da sua máquina.
- **Fora do escopo da etapa (como na especificação):** regras de aprovação configuráveis, conjuntos criados pela organização, troca de dono, exclusão de organização e recuperação de senha.

## Perguntas para você (Paulo) — respondidas em 26/11/2026
1. **Quem dá acessos pode dar o que não tem?** Resposta: sim. Só administradores dão acessos: o administrador da plataforma ou o administrador da organização (o dono e quem tem "Convidar pessoas e dar acessos"). O administrador da organização pode conceder qualquer alçada. Comportamento mantido.
2. **Risco aceito no convite?** Resposta: sim, risco aceito até existir a confirmação de e-mail.

## Ajuste de 26/11/2026 (critério B11)
- **O que mudou:** o administrador da plataforma passa a cuidar das pessoas de **qualquer** organização, mesmo sem ser membro dela. Em Administração → **Organizações** ele vê a lista de organizações; ao abrir uma, usa a mesma tela Pessoas (convidar, marcar alçadas, remover, cancelar e reenviar convites, ver o histórico).
- **Regras que continuam valendo:** o dono não pode ser alterado nem removido; ninguém altera as próprias alçadas; os limites de convites valem por organização. Quem não é administrador da plataforma é recusado pelo servidor em todas essas rotas.
- **Histórico:** cada ação aparece com o administrador da plataforma como autor, marcada "via administração da plataforma".
- **Convite feito pelo administrador da plataforma:** se ele deixar de ser administrador antes do aceite, o link deixa de valer.
- **Testes (26/11/2026, branch `etapa-2b-acessos` integrada):** `go vet` sem avisos, formatação ok, `go test` aprovado com cobertura de 72,7%; verificação de tipos e montagem aprovadas; Playwright **9 de 9 aprovados** (inclui o novo teste da tela Organizações). Revisões geral e de segurança: nenhum problema bloqueante; o ponto importante (teste do convite após perder o acesso de administrador) foi corrigido.
- **Ficou de fora (pontos menores, sem risco):** ao retirar o acesso de administrador de alguém, os convites que essa pessoa criou continuam aparecendo como pendentes, mas o link é recusado no aceite.

## Relacionado
[[Especificação - Etapa 2B]] · [[Roteiro de Construção - Fase 1A]] · [[PRD - Permissões e Aprovações]] · [[Índice - Projetos]]
