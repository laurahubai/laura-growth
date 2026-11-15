---
tipo: prd
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# PRD — Fase 1 — MVP de Posts

Parte do [[PRD - Plataforma Laura Growth]]. Fases: [[PRD - Fases e Escopo]].

## Objetivo da fase
Você ganhar tempo e **manter a cadência de posts** da Laura Growth e da Laura Hub, enquanto mede o custo real de cada atividade.

## Escopo
| | Fase 1A | Fase 1B |
|---|---|---|
| Redes | Instagram, Facebook, LinkedIn | + TikTok |
| Formatos | Imagem com legenda, carrossel | + vídeo curto |

## Fluxo de criação de um post
Em cada etapa o agente propõe e a pessoa pode **aprovar, pedir ajuste (com comentário), editar à mão ou descartar**.

| Etapa | O que o agente faz | Intervenção humana |
|---|---|---|
| 1. Pauta | Sugere tema, objetivo, rede e formato | Escolhe ou muda o tema |
| 2. Texto | Escreve legenda, título e hashtags | Ajusta o texto |
| 3. Visual | Cria a arte, o carrossel ou o vídeo | Pede nova versão ou envia material próprio |
| 4. Revisão | Confere marca, ortografia e formato de cada rede | Vê os avisos |
| 5. Aprovação final | Apresenta o post como ficará em cada rede | **Aprova ou devolve** (obrigatória) |
| 6. Publicação | Agenda e publica; confirma se foi ao ar | Pode reagendar ou cancelar |

**A definir:** quais etapas de 1 a 4 serão paradas obrigatórias e quais o agente pode seguir sozinho. Recomendação: na fase 1, somente a etapa 5 é obrigatória; nas demais, a pessoa intervém quando quiser.

## Requisitos

### Kit de marca
| Código | Requisito |
|---|---|
| MKT-01 | Cadastrar por organização: logo, cores, fontes, tom de voz, temas, público, o que nunca dizer e exemplos de posts. |
| MKT-02 | Enviar material avulso (fotos, vídeos, ideias, links) para um post específico. |
| MKT-03 | Os agentes usam sempre o kit da organização ativa, nunca o de outra. |

### Criação e intervenção
| Código | Requisito |
|---|---|
| POST-01 | Iniciar um post de duas formas: sugestão do agente ou pedido da pessoa. |
| POST-02 | Seguir o fluxo de seis etapas acima, mostrando em qual etapa cada post está. |
| POST-03 | Em qualquer etapa, permitir aprovar, pedir ajuste com comentário, editar à mão ou descartar. |
| POST-04 | Guardar as versões anteriores e o histórico de quem pediu o quê. |
| POST-05 | Adaptar o mesmo post ao formato de cada rede (tamanho da imagem, limite de texto). |
| POST-06 | Criar imagem única e carrossel (1A); vídeo curto (1B). |

### Aprovação
| Código | Requisito |
|---|---|
| APR-01 | Nenhum post é publicado sem aprovação final de uma pessoa. |
| APR-02 | Avisar a pessoa quando houver post aguardando (canal do aviso: a definir). |
| APR-03 | Post não aprovado até a data prevista **não** é publicado; fica marcado como atrasado. |

### Calendário e publicação
| Código | Requisito |
|---|---|
| CAL-01 | Calendário por organização com os posts e seus estados (em criação, aguardando aprovação, agendado, publicado, falhou). |
| CAL-02 | Definir a cadência desejada por rede (quantidade por semana: a definir por organização). |
| CAL-03 | O agente avisa quando o calendário está abaixo da cadência definida. |
| PUB-01 | Publicar na data e hora agendadas nas redes conectadas. Ver [[PRD - Conexões com Plataformas]]. |
| PUB-02 | Se a publicação falhar, avisar e permitir tentar de novo. |
| PUB-03 | Enquanto uma rede não estiver conectada, entregar o post pronto para publicação manual (arquivo e texto). |

### Medição de custo
| Código | Requisito |
|---|---|
| CST-01 | Registrar tokens e custo em dólar de cada etapa de cada post. Ver [[PRD - Painel de Custos e Créditos]]. |

## Fora desta fase
Convidados e permissões, anúncios pagos, relatórios de resultado, alertas de regras do setor, cobrança, inglês.

## A fase está pronta quando
- [ ] Laura Growth e Laura Hub existem como duas organizações separadas.
- [ ] Um post passa pelas seis etapas e é publicado nas redes da fase 1A.
- [ ] Você consegue intervir em qualquer etapa.
- [ ] O custo de cada post aparece no painel.
- [ ] A cadência definida é cumprida por um período de teste (duração a definir).

Passo a passo: [[Roteiro de Construção - Fase 1A]]. Agentes envolvidos: [[PRD - Agentes de IA]]. Medidas: [[PRD - Indicadores de Sucesso]].
