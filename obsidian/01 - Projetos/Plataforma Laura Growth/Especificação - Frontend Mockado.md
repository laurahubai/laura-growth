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
| Aprovações | Fila para aprovar ou devolver vários posts em sequência | APR-01, APR-04 |
| Posts | Lista de posts (com pilar e objetivo) e botão "Novo post" | POST-01, POST-02, POST-07 |
| Post (detalhe) | As 6 etapas, os 4 comandos, versões, atividade ao vivo, pilar, objetivo, CTA e piloto automático (com botão de pausar) | POST-01..05, POST-07..09, APR-01, PUB-03 |
| Calendário | Mês com os posts e seus estados; botão "Planejar semana com o agente" | CAL-01, CAL-04, APR-03 |
| Kit de marca | Cadastro do kit da organização ativa, com pilares de conteúdo, site e perfis | MKT-01, MKT-03, MKT-04 |
| Nova marca | Onboarding em 4 passos: sobre a marca, revisar o kit rascunhado pelo agente, cadência, plano da primeira semana | MKT-05, CAL-04 |
| Conexões | Estado das redes (no mock: todas "não conectadas", entrega manual) | CNX-02, CNX-06 |
| Custos | Visão Operação: tokens e dólares, custo médio, exportar planilha | CST-01..07 |
| Configurações | Cadência por rede, piloto automático (liga e desliga), cadastro das APIs de IA, API usada em cada etapa e restaurar dados | CAL-02, POST-09, AGT-07, AGT-08 |

Seletor de organização e botão "Nova marca" na barra superior (ORG-03).

## Paradas obrigatórias — decidido em 2026-11-15
Só a **etapa 5 (aprovação final)** é obrigatória. Com o **piloto automático** ligado (padrão), o agente segue sozinho pelas etapas 1 a 4. A pessoa pode pausar o piloto em um post para revisar etapa por etapa, ou desligá-lo em Configurações.

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

### J6 — Automação do posicionamento (incluído em 2026-11-15)
Cobre as sete sugestões aprovadas em [[Processo de Posicionamento Digital - Análise e Sugestões]].
- [ ] Em "Nova marca", o agente rascunha o kit e eu corrijo antes de concluir.
- [ ] O onboarding termina com a cadência definida e o plano da primeira semana.
- [ ] No kit, edito os pilares de conteúdo (de 3 a 5).
- [ ] No calendário, peço o plano da semana, ajusto e aprovo de uma vez.
- [ ] Aprovar o plano cria os posts, mas nenhum fica aprovado para publicação.
- [ ] Com o piloto automático, o post chega sozinho à aprovação final; consigo pausar no meio.
- [ ] Cada post mostra pilar, objetivo (atrair, engajar ou converter) e CTA.
- [ ] Em "Aprovações", aprovo ou devolvo um post após o outro na mesma tela.
- [ ] Em Configurações, cadastro uma API de IA (com chave fictícia) e vejo a chave mascarada.
- [ ] Escolho uma API para uma etapa, crio um post e vejo essa API no painel de Custos.

**Limites do protótipo:** nenhuma API de IA é chamada de verdade e a chave digitada é descartada ao salvar (ficam só os 4 últimos caracteres; não usar chaves reais); o agente não lê site nem perfis de verdade (o rascunho do kit é um exemplo); o plano é semanal, não mensal; temas e textos são modelos fixos.

### J4 — Calendário e cadência
- [ ] Vejo o mês com os estados dos posts.
- [ ] Vejo um post atrasado e o aviso de cadência abaixo do planejado.

### J5 — Custos
- [ ] Vejo tokens e dólares por organização, post, etapa e formato.
- [ ] Vejo o custo médio por post, separando publicados e descartados.
- [ ] Exporto a planilha.

## Depois da homologação
Registrar ajustes aqui e atualizar o [[Roteiro de Construção - Fase 1A]] antes do backend.
