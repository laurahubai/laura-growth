---
tipo: projeto
status: aprovado
area: Produto e Agentes de IA
criado: 2026-11-15
---
# Processo de Posicionamento Digital — Análise e Sugestões

Parte do [[PRD - Plataforma Laura Growth]]. Análise feita em **2026-11-15**, cruzando a ideia do fundador com a [[PRD - Fase 1 - MVP de Posts]], o [[Roteiro de Construção - Fase 1A]] e as telas da [[Especificação - Frontend Mockado]].

> A decisão de foco e as **sete sugestões** foram aprovadas pelo fundador em 2026-11-15. Tudo continua **planejado**, nada construído.

## Decisão de foco (aprovada)
Até **10/01/2027** a plataforma faz **somente posts**, para a Laura Hub e a Laura Growth, como máquina de automação de **posicionamento digital**. O processo de growth orientado por métricas será implementado depois dessa data. Registrada em [[PRD - Decisões de Negócio]].

## A ideia original e o veredito
O fundador listou três blocos que não podem faltar:
1. Onboarding de novas marcas e projetos
2. Calendário que possa ser automatizado
3. Criação, validação e postagem

**Veredito:** os três estão certos e são a espinha do processo. Sozinhos formam uma máquina de publicar. Growth é um ciclo completo: planejar → publicar → medir → aprender → ajustar. Medir e aprender ficam para depois de 10/01/2027.

## O ciclo completo e a situação atual
| Etapa | No PRD e no mockup hoje | O que falta |
|---|---|---|
| 1. Onboarding da marca | Tela "Kit de marca": formulário em branco com 7 campos | Objetivo do negócio, oferta, concorrentes, destino do interessado (site, WhatsApp) |
| 2. Estratégia | Não existe | Pilares de conteúdo e proporção entre atrair, engajar e converter |
| 3. Calendário | Mostra o mês e avisa quando falta post (CAL-03) | Só exibe, não planeja. Posts nascem um a um, sem horário |
| 4. Criação e validação | 6 etapas, 4 comandos, versões: bem resolvido | Chamada para ação e, mais tarde, link de destino |
| 5. Postagem | Manual no mock; automática na etapa 8 do roteiro | Nada de essencial |
| 6. Medição | Só na fase 3 | Resultado por post (depois de 10/01/2027) |
| 7. Aprendizado | Não existe | Agente usar o que funcionou para planejar (depois de 10/01/2027) |

**Outros pontos encontrados**
- O PRD tem organização (a marca), mas não tem **campanha**: grupo de posts com objetivo e período, como um lançamento.
- O calendário do mockup mostra só o mês; o roteiro pede também a semana. O post tem data, mas não tem hora (PUB-01 pede data e hora).
- A cadência fica em Configurações, separada do kit e das conexões. No onboarding, os três deveriam estar em sequência.

## Sugestões aprovadas para a fase 1A (até 10/01/2027)
Aprovadas em 2026-11-15. Viraram requisitos na [[PRD - Fase 1 - MVP de Posts]] e entraram no [[Roteiro de Construção - Fase 1A]].

| # | Sugestão | Por quê | Requisito |
|---|---|---|---|
| 1 | **Plano semanal em lote**: o agente propõe a semana inteira, respeitando cadência e pilares, e a pessoa aprova o plano em uma tela | É o que mais automatiza. Hoje é preciso clicar "Novo post" para cada um | CAL-04 |
| 2 | **Onboarding assistido**: a pessoa informa site e perfis da marca, o agente rascunha o kit e ela só corrige. Termina com a cadência definida e o plano do primeiro mês | Evita o formulário em branco | MKT-05 |
| 3 | **Pilares de conteúdo no kit** | Posicionamento é repetir os mesmos 3 a 5 assuntos com a mesma voz | MKT-04 |
| 4 | **Fila de aprovação**: aprovar vários posts em sequência, com aviso de pendência (canal do aviso: a definir, APR-02) | Sem ela, a aprovação post a post vira o gargalo | APR-04 |
| 5 | **Piloto automático nas etapas 1 a 4**: o agente segue sozinho até a aprovação final, que continua obrigatória | Era a recomendação do PRD; agora confirmada | POST-09 |
| 6 | **CTA em cada post**, como campo de texto simples, sem link rastreável | Todo post deve dizer o que a pessoa faz em seguida | POST-08 |
| 7 | **Guardar pilar e objetivo em cada post** (atrair, engajar ou converter) | Quando as métricas entrarem, já haverá histórico para comparar | POST-07 |

## Para depois de 10/01/2027
- Link rastreável em cada post
- Leitura de resultados por post (alcance, salvamentos, cliques)
- Revisão mensal automática: o agente mostra o que rendeu mais e propõe o ajuste do plano
- Reaproveitamento de posts que foram bem
- Responder comentários e mensagens; nutrição de leads

## Como medir sucesso até 10/01/2027
Só por **constância**: posts publicados em relação à cadência definida, como já previsto para a fase 1A em [[PRD - Indicadores de Sucesso]].

## Ponto de atenção
O objetivo declarado inclui **captação de lead**, mas o PRD mede interessados só na fase 3 e deixa WhatsApp fora das fases 1 a 3 ([[PRD - Fases e Escopo]]). Rever esse encaixe ao planejar o que vem depois de 10/01/2027.

## Termos
| Termo | Significado simples |
|---|---|
| **Pilar de conteúdo** | Um dos poucos assuntos fixos sobre os quais a marca sempre fala |
| **CTA** | Chamada para ação: o que o post pede que a pessoa faça (salvar, comentar, clicar) |
| **Link rastreável** | Link marcado que mostra de qual post veio a visita |
| **Cadência** | Quantidade de posts por semana em cada rede |

## Relacionado
[[Índice - Projetos]] · [[Home]] · [[PRD - Agentes de IA]] · [[Strategy Agent]]
