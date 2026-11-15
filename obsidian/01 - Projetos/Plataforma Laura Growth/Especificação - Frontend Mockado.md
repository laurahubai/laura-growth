---
tipo: projeto
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# Especificação — Frontend Mockado

> **Frontend mockado** = telas reais que funcionam com dados de exemplo, sem servidor nem IA de verdade. Serve para **homologar a jornada** antes de construir o backend.
> Tudo aqui é **proposta** para validar usando o próprio mock.

Parte do [[PRD - Plataforma Laura Growth]]. Cobre a [[PRD - Fase 1 - MVP de Posts]] (fase 1A). Visual: [[Design System - Plataforma Laura Growth]].

## Onde fica
- Código: `Laura Growth/plataforma/` (fora do cofre).
- Como abrir: dentro de `plataforma/`, rodar `npm run dev` e acessar `http://localhost:3000`.
- Os dados ficam guardados **só no navegador**. O botão "Restaurar dados de exemplo" (em Configurações) volta ao início.

## Mapa de telas (menu) — proposta
| Tela | Para quê | Requisitos |
|---|---|---|
| Entrar | Acesso com e-mail e senha (qualquer valor no mock) | — |
| Início | O que precisa da minha atenção: aprovações, atrasos, cadência | APR-02, CAL-03 |
| Posts | Lista de posts e botão "Novo post" | POST-01, POST-02 |
| Post (detalhe) | As 6 etapas, os 4 comandos, versões e atividade ao vivo do agente | POST-01..05, APR-01, PUB-03 |
| Calendário | Mês com os posts e seus estados | CAL-01, APR-03 |
| Kit de marca | Cadastro do kit da organização ativa | MKT-01, MKT-03 |
| Conexões | Estado das redes (no mock: todas "não conectadas", entrega manual) | CNX-02, CNX-06 |
| Custos | Visão Operação: tokens e dólares, custo médio, exportar planilha | CST-01..07 |
| Configurações | Cadência por rede e restaurar dados | CAL-02 |

Seletor de organização na barra superior (ORG-03).

## Paradas obrigatórias — proposta
Só a **etapa 5 (aprovação final)** é obrigatória. Nas etapas 1 a 4 o agente propõe e a pessoa pode aprovar, pedir ajuste, editar à mão ou descartar.

## Estados do post — proposta
| Estado | Quando |
|---|---|
| Em criação | Etapas 1 a 4 |
| Aguardando aprovação | Na etapa 5 |
| Aprovado | Aprovado, sem data |
| Agendado | Aprovado com data futura |
| Pronto para publicação manual | Aprovado, rede não conectada (PUB-03) |
| Publicado | Marcado como publicado |
| Falhou | Erro de publicação (PUB-02) |
| Atrasado | Data passou sem aprovação (APR-03) |
| Descartado | Pessoa descartou (CST-05) |

## Tela ao vivo — proposta
Painel "Atividade do agente" no detalhe do post: os passos aparecem um a um enquanto o agente trabalha (simulado com pausas), no estilo do Claude Code. No backend real, virá do mecanismo ao vivo descrito em [[Tecnologias de Base]].

## Dados de exemplo
Organizações Laura Growth e Laura Hub; posts, textos, imagens, tokens e custos **fictícios** e marcados como exemplo. Fornecedor e preços de IA: a definir.

## Roteiro de homologação
### J1 — Acesso e organizações
- [ ] Entro com e-mail e senha.
- [ ] Troco entre Laura Growth e Laura Hub e cada uma mostra só os seus dados.

### J2 — Kit de marca
- [ ] Edito o kit de uma organização e salvo.
- [ ] O kit não aparece na outra organização.

### J3 — Criar um post
- [ ] Crio um post por pedido meu e outro aceitando sugestão de pauta.
- [ ] Vejo o agente trabalhando ao vivo.
- [ ] Peço ajuste com comentário e recebo nova versão; versões antigas ficam no histórico.
- [ ] Edito o texto à mão.
- [ ] Vejo a prévia por rede e aprovo; o post não fica aprovado sem minha ação.
- [ ] Baixo o post pronto para publicação manual.

### J4 — Calendário e cadência
- [ ] Vejo o mês com os estados dos posts.
- [ ] Vejo um post atrasado e o aviso de cadência abaixo do planejado.

### J5 — Custos
- [ ] Vejo tokens e dólares por organização, post, etapa e formato.
- [ ] Vejo o custo médio por post, separando publicados e descartados.
- [ ] Exporto a planilha.

## Depois da homologação
Registrar ajustes aqui e atualizar o [[Roteiro de Construção - Fase 1A]] antes do backend.
