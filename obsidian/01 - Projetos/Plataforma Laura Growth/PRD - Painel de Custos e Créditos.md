---
tipo: prd
status: rascunho
area: Comercial
criado: 2026-11-15
---
# PRD — Painel de Custos e Créditos

Parte do [[PRD - Plataforma Laura Growth]].

## Por que existe
A cobrança será por **créditos**, mas os planos só serão criados depois de conhecer o **custo operacional real**. Este painel é a fonte desses números. Por isso entra já na fase 1A.

## Duas visões
| Visão | Quem vê | O que mostra | Fase |
|---|---|---|---|
| **Operação** | Só a equipe Laura | Tokens e custo em dólar | 1A |
| **Organização** | Dono e quem ele autorizar | Apenas créditos consumidos | 2 (provisório) · 4 (definitivo) |

A organização **nunca** vê tokens nem dólares.

## Requisitos — visão Operação
| Código | Requisito |
|---|---|
| CST-01 | Registrar, a cada tarefa de agente: organização, agente, atividade (etapa do fluxo), post ou campanha, fornecedor de IA, quantidade de tokens, custo em dólar, data. |
| CST-02 | Registrar também custos que não são medidos em tokens (ex.: imagem e vídeo costumam ser cobrados por unidade ou por segundo), sempre convertidos em dólar. |
| CST-03 | Mostrar totais por: organização, campanha, post, atividade, agente, formato (imagem, carrossel, vídeo), rede e período. |
| CST-04 | Mostrar o **custo médio por post** por formato, incluindo as versões descartadas e os ajustes pedidos. |
| CST-05 | Separar o custo de posts publicados do custo de posts descartados. |
| CST-06 | Exportar os dados em planilha. |
| CST-07 | Guardar o preço usado em cada registro, pois os fornecedores mudam preços ao longo do tempo. |

## Requisitos — visão Organização
| Código | Requisito | Fase |
|---|---|---|
| CRD-01 | Mostrar créditos consumidos por período, campanha e tipo de atividade. | 2 |
| CRD-02 | Enquanto a conversão não estiver definida, marcar o número como **provisório**. | 2 |
| CRD-03 | Saldo, limite e aviso de créditos acabando. | 4 |
| CRD-04 | Compra de créditos e planos. | 4 |

## A definir (depende do teste prático)
| Item | Situação |
|---|---|
| Quanto vale 1 crédito | a definir |
| Regra de conversão de custo em créditos | a definir |
| Planos e preços | a definir |
| Moeda de cobrança em cada mercado | a definir |
| Se ajustes e versões descartadas consomem créditos | a definir |

## Perguntas que o painel precisa responder ao final do teste
1. Quanto custa, em média, um post de cada formato?
2. Quanto custa manter a cadência de uma organização por mês?
3. Qual atividade mais pesa no custo?
4. Quanto do custo vem de retrabalho (ajustes e descartes)?

Relacionado: [[Modelo Comercial]] · [[PRD - Indicadores de Sucesso]]
