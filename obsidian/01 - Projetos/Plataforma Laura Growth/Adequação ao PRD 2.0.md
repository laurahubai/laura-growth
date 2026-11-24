---
tipo: projeto
status: aprovado
area: Produto e Agentes de IA
criado: 2026-11-24
---
# Adequação ao PRD 2.0

> **Incorporada ao PRD 2.0 v1.1** em 24/11/2026 ([[prd-lauragrowth20|PRD LauraGrowth 2.0]], seção "Histórico de alterações").

## Objetivo
Adaptar a plataforma já construída (etapas 1 e 2) ao [[prd-lauragrowth20|PRD LauraGrowth 2.0]], que **substitui o PRD atual** ([[PRD - Plataforma Laura Growth]]) e aproveita o que já existe. Não é recomeço.

## Entrega final e prazo
- MVP **até a publicação automática**. Colher resultados, aprender e melhorar ficam para depois.
- Usos: (1) sua plataforma para lançar a **Laura Hub em 10/01/2027**; (2) depois, plataforma para os clientes da Laura Hub.
- Prazo de cada etapa: a definir. Todo o desenvolvimento será feito por agentes de IA.

## Etapas
A ordem nova está no [[Roteiro de Construção - Fase 1A]] (etapas 2A a 2E, antes da etapa 3):
1. **2A** — Cadastro de IA: chave de API ou assinatura logada, com tokens e US$.
2. **2B** — Acessos por pessoa e por alçada.
3. **2C** — O kit de marca vira o DNA da marca.
4. **2D** — Chat do CMO (no lugar do onboarding de 4 passos).
5. **2E** — Plano de marketing com aprovação.
6. Etapas 3 a 8 do roteiro (posts, imagem, aprovação, calendário, custos, publicação automática).

## Decisões (2026-11-24)
| Tema | Decisão |
|---|---|
| PRD 2.0 | Substitui o PRD atual, com adequação do que já foi construído |
| Quem usa | Cliente, equipe Laura Hub e agência do cliente |
| Setores do MVP | A própria plataforma (Laura Hub e Laura Growth) e clínicas de estética. Outros setores só se surgir cliente, sem data |
| Fornecedores de IA | No mínimo 2 por serviço, cadastrados dentro da plataforma (chave, limites, tokens) |
| Tipo de cadastro de IA | Chave de API **ou** assinatura logada (ex.: Claude Code, Codex, Antigravity) em alguma etapa |
| Custo | Toda atividade registra tokens e US$. Na assinatura o US$ é **estimado** (equivalente ao preço da API) e o painel mostra isso |
| Assinatura logada | Só para as organizações da casa (Laura Hub e Laura Growth); clientes usam chave de API. *Conferir os termos de uso de cada fornecedor antes de usar* |
| Acessos | Por pessoa e por alçada, **sem perfis fixos** |
| Onboarding | Chat IA ↔ humano: o CMO e a pessoa constroem as respostas juntos. O onboarding atual (protótipo) é a base |
| Mínimo para seguir | Campos "essenciais" do PRD 2.0 + o que já existe no kit |
| Qualidade do plano | Aprovação humana |
| Reaprovação | Qualquer mudança em post aprovado o devolve para aprovação |
| Regras do setor | Entram no onboarding como "o que evitar". **Bloqueia** só o que o cliente marcou em "o que evitar"; regras do setor continuam **só alerta** |
| Ordem | O chat do CMO pode vir antes dos posts (há tempo) |

## Comparação: o que existe x PRD 2.0
Base: código em `sistema/` (último registro 8359c52, de 16/11/2026) e protótipo aprovado ([[Especificação - Frontend Mockado]]).

| Parte do PRD 2.0 | Situação | O que fazer |
|---|---|---|
| Separação entre marcas | Reaproveitar | Já existe e foi testada. Cada organização é uma marca |
| Setores | Reaproveitar | O "tipo de organização" já tem empresa e clínica de estética |
| Brand DNA (RF-03) | Ampliar | Kit atual serve de base. Faltam campos, versões e origem de cada informação |
| Brand Brain (RF-10) | Ampliar | Nasce do kit: cada informação com origem, status (confirmada ou sugerida) e versão |
| Perfis e permissões (§3) | Ampliar | Hoje só "dono" e "membro". Passa a ser por alçada |
| CMO e onboarding (RF-01, §7) | Novo | Os 4 passos do protótipo viram chat, com progresso ao lado |
| Cadastro de IA (§8) | Já planejado | Estava na etapa 3. Passa a ser a etapa 2A, com a opção de assinatura |
| Plano de marketing (RF-08) | Ampliar | Era o plano semanal. Passa a ter objetivo, canais, pilares e 30 dias |
| Posts, Guardian, aprovação (RF-11, RF-12) | Já planejado | Etapas 3 a 5 |
| Agenda (RF-09) | Já planejado | Etapa 6 |
| Publicação (RF-13) | Já planejado | Etapa 8 |
| Anúncios, métricas, outros setores, vídeo | Fora do MVP | — |

## Pendências
- [ ] Contratar ou definir o primeiro fornecedor de IA (bloqueia testes reais do chat e dos posts).
- [ ] Conferir os termos de uso das assinaturas (Claude Code, Codex, Antigravity) para uso pela plataforma.
- [ ] Confirmar a lista de alçadas (proposta na etapa 2B do roteiro).
- [ ] Confirmar se mudar só a data ou o horário de um post aprovado também exige nova aprovação.
- [x] Atualizar o PRD 2.0 com estas decisões (publicação automática no MVP, setores, acessos, assinaturas).

## Relacionado
[[Índice - Projetos]] · [[PRD - Plataforma Laura Growth]] · [[PRD - Decisões de Negócio]] · [[Roteiro de Construção - Fase 1A]] · [[Home]]
