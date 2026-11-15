---
tipo: benchmark
status: rascunho
area: produto
criado: 2026-11-15
---
# Benchmark - Crush AI

> Estudo de um concorrente de referência: **Crush AI** (trycrush.ai), um "gestor de anúncios com IA" para Meta (Facebook e Instagram).
> Fonte: o site público e a demonstração do produto na home, consultados em 2026-11-15. A demo usa uma loja inventada (Marea, "Vestido Brisa") e **todos os números são exemplos** do próprio site. Não testamos o produto por dentro.

Relacionado: [[Referência UX-UI - Crush AI]] · [[PRD - Plataforma Laura Growth]] · [[PRD - Agentes de IA]] · [[PRD - Permissões e Aprovações]] · [[PRD - Painel de Custos e Créditos]]

## Estrutura das telas
- **Barra lateral:** seletor de produto/loja · Home · Library (biblioteca) · Ads (anúncios) · Channels (canais): Meta ads (LIVE), Autopilot (ligado/desligado), Organic · saldo de **créditos** e usuário no rodapé.
- **Home:** saudação com a IA em primeira pessoa ("Desde ontem fiz 2 anúncios e gastei $14,10 para 2 vendas"); 4 números (gasto, vendas, custo por venda, valor vendido); faixa "esta semana: 3 testes concluídos, 1 vencedor"; gráfico de vendas por semana; novidades da biblioteca.

## O fluxo de trabalho (6 etapas)
1. **Find (encontrar):** a biblioteca reúne anúncios de concorrentes, modelos e uploads. O destaque é **há quantos dias o anúncio do concorrente está no ar** (se continua rodando, provavelmente dá lucro).
2. **Make (criar):** a IA cria versões próprias inspiradas nesses anúncios, mostrando a origem. Cada uma passa por um **"Crítico"** (um segundo agente que revisa a qualidade). Nada vai para a Meta sem o OK do cliente.
3. **Test (testar):** 1 clique cria o teste na conta de anúncios do próprio cliente (3 conjuntos, verba igual). A publicação começa à meia-noite para a Meta distribuir o orçamento do dia inteiro.
4. **Judge (julgar):** regras fixas e visíveis depois de 48 h:
   - **Vencedor:** 3 ou mais vendas no custo-alvo ou abaixo dele.
   - **Pausa imediata:** gastou 2× o alvo sem nenhuma venda.
   - **Os demais:** ganham mais 24 h.

   A verba do anúncio pausado vai para o vencedor.
5. **Autopilot (piloto automático):** repete o ciclo sozinho dentro de um **teto diário**. Modos "Pedir antes" ou "Sem intervenção"; botão "Pausar tudo"; **registro de decisões** em linguagem simples; fila de aprovação ("Lançar / Agora não").
6. **Organic (novo):** um post por dia, o mesmo para Instagram, Facebook, YouTube e TikTok, com horário por rede e resultado por canal. Sem gasto com anúncio.

## Funil de venda (site)
Página de vendas longa → preços com promoção e contagem regressiva → checkout em janela sobre a página. O cliente **paga antes** de criar a conta e conectar o Facebook. Planos de 2, 4 ou 8 semanas que renovam automaticamente por um valor mensal mais alto.

## O que vale levar para a Laura Growth
- **O ciclo fechado** encontrar → criar → testar → julgar → escalar, com uma tela para cada etapa. O piloto automático é esse ciclo rodando sozinho.
- **Regras de decisão claras e visíveis.** Geram confiança sem exigir que o cliente entenda de anúncios.
- **Controle do cliente:** teto de gasto, "pedir antes", "pausar tudo" e histórico de decisões. Ver [[PRD - Permissões e Aprovações]].
- **A IA falando em primeira pessoa e um agente "Crítico" revisando outro**, o que combina com a ideia de "time de agentes". Ver [[PRD - Agentes de IA]].
- **Créditos** como forma de cobrança pelo uso. Ver [[PRD - Painel de Custos e Créditos]].
- **Evitar sem antes avaliar:** a renovação automática por um valor mais alto e as promoções com contagem regressiva, que podem gerar reclamações.

> Estas são sugestões de estudo, **a confirmar**. Nada aqui é decisão aprovada.

## Encaixe com a proposta da Laura Growth
Análise feita em 2026-11-15 a partir do [[PRD - Plataforma Laura Growth]], das [[PRD - Decisões de Negócio]] e do [[Laura Growth - Posicionamento]].

**Resumo:** encaixe forte nos princípios e na forma de trabalhar; mais fraco no público e na forma de medir resultado. Usar o Crush como **referência de fluxo e telas**, não como modelo de produto.

### Onde há encaixe
| Crush | Laura Growth (PRD) |
|---|---|
| A IA faz e a pessoa aprova ("pedir antes", fila de aprovação) | "IA executa, humano aprova", com aprovação configurável |
| Créditos como cobrança | Créditos (decidido) |
| "Crítico" revisando o que a IA criou | Etapa 4 do fluxo (Revisão) em [[PRD - Fase 1 - MVP de Posts]] |
| Organic: um post por dia em várias redes | Fase 1A (MVP) |
| Ciclo testar → julgar → escalar anúncios | Fase 3 (Ads Agent e Performance Agent) |
| IA em primeira pessoa e registro de decisões | Reforça o "time de marketing com IA" |

### Onde não encaixa
- **Público:** o Crush atende lojas online, com o cliente fazendo tudo sozinho. O nosso primeiro cliente é a clínica de estética média, com um responsável de marketing e terceiros convidados.
- **Como medir resultado:** o Crush mede venda e custo por venda. A clínica recebe contatos e agenda consultas. A contagem de agendamentos fica para fase posterior, então as regras de "vencedor ou pausa" não se aplicam como estão.
- **Regras do setor:** anúncio de estética tem restrições da Meta e dos conselhos de saúde. Um piloto automático agressivo combina mal com isso. Ver [[PRD - Conformidade e Regras do Setor]].
- **Nosso diferencial, que o Crush não tem:** a clínica dona das contas e do histórico, a troca de agência sem perder nada e o convite de terceiros com permissões.

### O que levar, por fase (a confirmar)
- **Fase 1A:** referência de posts de concorrentes que estão dando resultado na etapa "Pauta" (no estilo da aba Find) e uma Home com resumo em primeira pessoa ("esta semana preparei 3 posts, 1 aguarda você"). As duas cabem no fluxo atual sem mudar decisões.
- **Fase 3:** regras de decisão claras e visíveis, teto de gasto e botão "pausar tudo", com a métrica em custo por contato ou por agendamento.
