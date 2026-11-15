---
tipo: decisões
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# PRD — Decisões de Negócio

Parte do [[PRD - Plataforma Laura Growth]]. Decisões tomadas na jornada de perguntas e respostas de **2026-11-15**. Decisões do projeto de lançamento ficam em [[Registro de Decisões]].

## Quem usa e quem paga
| Tema | Decisão |
|---|---|
| Quem pode pagar | Todos: clínica com time interno, agência que atende clínica, clínica que paga acesso completo e convida terceiros |
| Relação clínica e prestador | Todos os modelos: prestador gerencia clínicas, marketplace, clínica sozinha |
| Primeiro cenário a funcionar | Clínica que convida terceiros |
| Dono dos dados, contas e conteúdos | Sempre a clínica |
| Perfil da clínica | Média, com um responsável de marketing |
| Segmento | Estética é o primeiro; outros depois |

## Como funciona
| Tema | Decisão |
|---|---|
| Papel da IA | IA executa, humano aprova |
| Aprovação final | Parametrizável por clínica |
| Permissões dos convidados | Sob medida, por pessoa |
| Regras de publicidade do setor | Só alerta, sem bloquear |
| Frentes da primeira versão | Redes sociais, anúncios pagos, relatórios |
| Plataformas conectadas | Instagram e Facebook, Google Ads, TikTok, Google Meu Negócio |
| Medição de interessados e agendamentos | Dados das redes e dos anúncios |

## Comercial
| Tema | Decisão |
|---|---|
| Cobrança | Por créditos. Primeiro medir tokens e custo em dólar por atividade e campanha; depois criar os planos |
| Painel de custo | Laura vê tokens e dólares; clínica vê só créditos |
| Serviço humano da Laura | Opção paga, entrando como terceiro convidado |
| Laura Hub | Produto separado |
| Mercados | Brasil, EUA e Europa |
| Idiomas | Português e inglês |

## Construção e teste
| Tema | Decisão |
|---|---|
| Quem constrói | O fundador, com ferramentas de IA |
| Ferramenta de construção | Claude Code (decidido em 2026-11-15) |
| Prioridade | MVP de automação de posts, o quanto antes |
| Foco até 10/01/2027 | Somente posts da Laura Hub e da Laura Growth, como máquina de automação de posicionamento digital. Growth orientado por métricas fica para depois dessa data (decidido em 2026-11-15; ver [[Processo de Posicionamento Digital - Análise e Sugestões]]) |
| Primeiro teste | Marketing da Laura Growth e da Laura Hub; depois uma clínica parceira real |
| Divisão em fases | Aprovada, incluindo 1A (imagem e carrossel) e 1B (vídeo e TikTok) |
| Cor da marca na plataforma | Coral própria (diferente do azul da Laura Hub), com botões principais em grafite (decidido em 2026-11-15; ver [[Benchmark - Cor de Marca]]) |
| Tecnologias | Mesmas da Laura Hub (Next.js, Go, PostgreSQL, River, ponte Python); tela atualizada ao vivo durante o trabalho da IA (decidido em 2026-11-15; mecanismo SSE a confirmar — ver [[Tecnologias de Base]]) |
| Fonte da plataforma | Inter (decidido em 2026-11-15; ver [[Design System - Plataforma Laura Growth]]) |

## MVP de posts
| Tema | Decisão |
|---|---|
| Redes | Instagram, LinkedIn, Facebook, TikTok |
| Formatos | Imagem com legenda, carrossel, vídeo curto |
| Material | Kit de marca como base, mais material avulso |
| Aprovação | Post a post, com intervenção humana durante a criação |
| Paradas obrigatórias | Só a aprovação final (etapa 5); nas etapas 1 a 4 o agente segue sozinho (decidido em 2026-11-15) |
| Automação aprovada para a fase 1A | Plano semanal em lote, onboarding assistido, pilares de conteúdo, pilar e objetivo em cada post, fila de aprovação, piloto automático, CTA em cada post (decidido em 2026-11-15; ver [[Processo de Posicionamento Digital - Análise e Sugestões]]) |

## Evolução em relação ao conceito original
- **Antes:** "agência de IA que executa marketing para o cliente" ([[Laura Growth - Posicionamento]]).
- **Agora:** plataforma em que cada organização tem seu espaço, os agentes executam e a organização convida quem quiser, inclusive a própria Laura como prestadora.
- As notas de conceito não foram reescritas. Revisar o posicionamento e o [[Modelo Comercial]] quando esta direção for confirmada.

## Ajustes propostos pelo PRD (a confirmar)
1. Tratar o cliente como **organização**, pois o primeiro uso é da própria Laura.
2. Deixar **inglês e os mercados dos EUA e Europa para a fase 4**.
3. Tratar **agendamentos** como medida de fase posterior, pois depende de integração com a agenda.
4. Mostrar créditos à clínica como **provisórios** até a conversão ser definida.
