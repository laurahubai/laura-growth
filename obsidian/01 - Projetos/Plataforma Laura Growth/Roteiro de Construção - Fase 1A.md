---
tipo: roteiro
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# Roteiro de Construção — Fase 1A

Parte do [[PRD - Plataforma Laura Growth]]. Detalha, em ordem, como construir o que está em [[PRD - Fase 1 - MVP de Posts]].

> Roteiro **planejado**. Datas e ferramentas: a definir.

> **Atualização 2026-11-24:** ordem revista para a adequação ao PRD 2.0. Entram as etapas **2A a 2E** antes da etapa 3; o cadastro de IA sai da etapa 3 e vai para a 2A. Motivos e decisões: [[Adequação ao PRD 2.0]].

## Como usar este roteiro
- Uma etapa por vez. Só passe para a próxima quando a lista "pronto quando" estiver toda marcada.
- Cada etapa termina em algo que você consegue **ver e testar**.
- Ao pedir a etapa para a ferramenta de IA, copie os requisitos citados (os códigos levam às notas do PRD).
- A partir da **etapa 5** a plataforma já é útil no dia a dia, mesmo sem publicação automática.
- **Regras de entrega (decididas em 25/11/2026):** cada etapa sai numa branch própria (ex.: `etapa-2a-1-cadastro-ia`), partindo da `main` atualizada, e vira um PR para você revisar; nunca push direto na `main` nem merge pelo agente. Vale o mínimo de pronto do [[prd-lauragrowth20|PRD 2.0]] §15.1 (um teste por critério, testes antigos passando, typecheck e `go vet` limpos, cobertura do servidor ≥ 60%, teste de tela com Playwright, migração que só acrescenta, backup antes de migrar). Na VM, dev e stage são a mesma pasta e o mesmo banco.

## Visão geral
| Etapa | Entrega | Já dá para usar? |
|---|---|---|
| 0 | Preparação | — |
| 1 | Acesso e organizações | Não |
| 2 | Kit de marca | Não |
| 2A-1 | Cadastro de IA por chave de API + registro de tokens e US$ | Não |
| 2A-2 | Cadastro de IA por assinatura logada (após conferir os termos) | Não |
| 2B | Acessos por pessoa e por alçada | Não |
| 2C | DNA da marca (kit ampliado, com versões) | Não |
| 2D | Chat do CMO (onboarding conversado) | Parcial (DNA da marca) |
| 2E | Plano de marketing com aprovação | Parcial (plano) |
| 3 | Pauta e texto com intervenção humana + registro de custo | Parcial (textos) |
| 4 | Imagem e carrossel | Parcial |
| 5 | Revisão, aprovação final e entrega manual | **Sim — primeiro marco** |
| 6 | Calendário e cadência | Sim |
| 7 | Painel de tokens e custos | Sim |
| 8 | Publicação automática | **Sim — fase 1A completa** |
| 9 | Período de teste | — |

---

## Etapa 0 — Preparação
Nada é construído aqui; são decisões e materiais que destravam o resto.

- [x] Escolher a ferramenta de IA para construir a plataforma: **Claude Code**.
- [x] Homologar a jornada no frontend mockado: **aprovado em 2026-11-15**. Ver [[Especificação - Frontend Mockado]].
- [x] Definir em qual pasta ficará o código da plataforma: máquina `gcp-lhac-compozyos-001`, pasta `/lhac/dev/laura-growth`. Código começa do zero; atualização ao vivo por SSE. Ver [[Tecnologias de Base]].
- [x] Definir as tecnologias de base: **mesmas da Laura Hub**, com atualização da tela ao vivo enquanto a IA trabalha. Ver [[Tecnologias de Base]]. *(Servidor de hospedagem ainda a definir.)*
- [ ] Escolher os fornecedores de IA para texto e para imagem *(a definir)*.
- [ ] **Verificar o processo de autorização** de LinkedIn, Facebook e Instagram para publicação automática e dar entrada o quanto antes. É o item com maior risco de atraso ([[PRD - Conexões com Plataformas]]).
- [ ] Reunir o material de marca da Laura Growth e da Laura Hub: logo, cores, fontes, tom de voz, temas, exemplos de posts.
- [ ] Definir a cadência desejada por rede para cada marca.
- [ ] Confirmar que as páginas das duas marcas existem nas três redes e que você é administrador.

## Etapa 1 — Acesso e organizações
**Constrói:** entrada com e-mail e senha; criação de organizações; troca entre elas.
**Requisitos:** ORG-01, ORG-02, ORG-03 ([[PRD - Modelo de Organizações]]).

Pronto quando:
- [x] Você entra na plataforma com sua conta.
- [x] Existem duas organizações: Laura Growth e Laura Hub.
- [x] Você troca de uma para a outra e cada uma mostra só o que é dela.

> **Concluída em 2026-11-16** na máquina de desenvolvimento (pasta `sistema/` do repositório). Verificado com testes automáticos (entrar, senha errada, troca de organização, recusa de organização alheia, sair, pedido de outro site, excesso de tentativas) e no navegador. Como rodar: `sistema/README.md`.

## Etapa 2 — Kit de marca
**Constrói:** tela para cadastrar e editar o kit de cada organização, incluindo os pilares de conteúdo; envio de arquivos; onboarding assistido, em que o agente rascunha o kit a partir do site e dos perfis da marca.
**Requisitos:** MKT-01, MKT-03, MKT-04, MKT-05.

Pronto quando:
- [ ] Os dois kits estão cadastrados com o material da etapa 0.
- [x] O kit de uma organização não aparece na outra.
- [ ] Cada kit tem de 3 a 5 pilares de conteúdo.
- [ ] Você informa site e perfis de uma marca, recebe um rascunho do kit e corrige antes de salvar. *(adiado: depende do fornecedor de IA, decidido em 2026-11-16)*

O plano do primeiro mês, que fecha o onboarding, passa a existir na etapa 2E (antes: etapa 6). O rascunho do kit por IA passa a ser feito pelo chat do CMO (etapa 2D).

> **Construído em 2026-11-16** (stage: https://stage-growth.laurahub.com.br): tela do kit com todos os campos do MKT-01, pilares (até 5, aviso abaixo de 3), cores, logo e arquivos de referência (PNG, JPG, WebP ou PDF, até 8 MB). Isolamento entre organizações verificado em testes automáticos e no navegador. **Falta:** cadastrar os dois kits com o material da etapa 0 (tarefa do Paulo) e o rascunho do kit por IA (adiado).

## Etapa 2A — Cadastro de IA
**Constrói:** cadastro de fornecedores de IA em Configurações, de dois tipos: **chave de API** ou **assinatura logada** (ex.: Claude Code, Codex, Antigravity). Escolha do fornecedor de cada etapa, com reserva. Registro de tokens e custo em US$ de toda atividade.
**Requisitos:** AGT-07, AGT-08, AGT-09, CST-01; PRD 2.0 §8 e §12 (custo).

Dividida em duas partes (decidido em 2026-11-24; [[Revisão PRD 2.0 - Etapa 2A]]). O cadastro vale para a plataforma toda e, por enquanto, só você cadastra.

### 2A-1 — Chave de API (começa já)
Testes com um **fornecedor simulado**, identificado como tal na tela e nos registros, e **uma chave real de texto** no teste final. Os 2 fornecedores reais por serviço passam a ser cobrados quando forem contratados.

Detalhes decididos em 25/11/2026 ([[prd-lauragrowth20|PRD 2.0]] v1.4, §3, §8.3, §9, §11): área de **Administração** só para o administrador da plataforma; cadastro de **modelos** com "Ler modelos disponíveis" e inclusão manual; endereço preenchido pelo tipo (OpenAI, compatível com OpenAI, Anthropic, simulado); preços manuais com vigência; limite mensal e percentual de aviso por fornecedor ou por modelo; principal e reserva obrigatórios; botão "Testar"; chave cifrada com `LG_CHAVE_CIFRAGEM` (já no `.env` da VM).

Pronto quando (critérios AC29, AC30 e AC32 a AC39 do PRD 2.0):
- [x] Só você vê e usa a área de Administração; outra pessoa é recusada pelo servidor (AC37).
- [x] Você cadastra fornecedores por chave de API para texto e imagem (simulado vale enquanto não houver contratação), com endereço preenchido pelo tipo; endereço sem https ou interno é recusado (AC39).
- [x] A chave não aparece inteira depois de salva (AC32).
- [x] "Ler modelos disponíveis" traz a lista do fornecedor, e você inclui um modelo à mão; falhas aparecem com mensagem clara (AC38).
- [x] Você cadastra preço por modelo, com vigência, e limite com percentual de aviso; o aviso aparece e, no limite, a reserva assume (AC34).
- [x] Você escolhe principal e reserva por atividade; sem os dois, a atividade não roda (AC36); quando o principal falha, a reserva conclui e o registro mostra qual atendeu (AC33).
- [x] "Testar" funciona para texto e imagem e grava tokens e US$, com o simulado marcado como **simulado** (AC29, AC35).
- [x] Organizações não veem nem cadastram fornecedores (AC30).
- [ ] Com a chave real de texto, um teste grava tokens e US$ reais (aguarda fornecedor contratado).

Situação em 25/11/2026: itens marcados comprovados por testes automáticos (Go e Playwright) na branch `etapa-2a-1-cadastro-ia`, aguardando o PR. Falta só o teste com a chave real.

### 2A-2 — Assinatura logada (depois de conferir os termos)
**Antes de começar:** conferir os termos de uso das assinaturas para uso pela plataforma e definir como a plataforma se conecta à assinatura e como conta os tokens.

Pronto quando:
- [ ] Você cadastra uma assinatura logada e a plataforma consegue usá-la em uma etapa.
- [ ] Assinatura logada só atende atividades das organizações da casa (Laura Hub e Laura Growth).
- [ ] Na assinatura, o valor em US$ aparece marcado como **estimado**.

A 2A-2 não bloqueia as etapas seguintes: a 2B pode começar quando a 2A-1 estiver pronta.

## Etapa 2B — Acessos por pessoa e por alçada
**Constrói:** convite de pessoas para uma organização e escolha, pessoa a pessoa, do que cada uma pode fazer. **Sem perfis fixos.**
**Requisitos:** [[PRD - Permissões e Aprovações]]; PRD 2.0 §3.

Pronto quando:
- [ ] Você convida uma pessoa por e-mail para uma organização.
- [ ] Você marca as alçadas dela uma a uma.
- [ ] Uma pessoa sem a alçada não consegue fazer a ação, nem forçando pelo endereço da página (verificação no servidor).

Lista de alçadas proposta *(a confirmar)*: ver a marca e os posts; conversar com o CMO e editar o DNA da marca; aprovar o plano de marketing; criar e editar posts; aprovar posts para publicação; conectar e desconectar redes; ver custos (US$ ou só créditos); convidar pessoas e dar acessos.

Permissão da equipe Laura (não é alçada da organização): cadastrar fornecedores de IA. Até a 2B, só o administrador da plataforma (decidido em 2026-11-24; [[prd-lauragrowth20|PRD 2.0]] §8.3).

## Etapa 2C — DNA da marca
**Constrói:** o kit de marca ampliado com os campos essenciais do PRD 2.0, versões e a origem de cada informação.
**Requisitos:** MKT-01, MKT-04; PRD 2.0 RF-02, RF-03, RF-10.

Pronto quando:
- [ ] O kit tem também: produtos ou serviços e qual priorizar, diferencial, modelo (B2B, B2C…), regiões, como os clientes compram, maior desafio, objetivo dos próximos 90 dias e canais escolhidos.
- [ ] "O que evitar" fica em destaque (inclui as restrições do setor informadas pelo cliente).
- [ ] Cada informação mostra se foi **confirmada por você** ou **sugerida pela IA**.
- [ ] Toda gravação gera uma versão; você vê as anteriores e quem mudou.
- [ ] Os dados já cadastrados no kit continuam lá.

## Etapa 2D — Chat do CMO
**Constrói:** o onboarding como conversa entre o CMO de IA e a pessoa, construindo as respostas juntos. Substitui o onboarding de 4 passos do protótipo, que vira a barra de progresso ao lado.
**Requisitos:** MKT-05; PRD 2.0 RF-01, §7 e §8.

Pronto quando:
- [ ] O CMO faz uma pergunta principal por vez e explica por que precisa dela quando útil.
- [ ] Quando você não sabe, ele propõe uma sugestão; ela entra no DNA como **sugerida** até você confirmar.
- [ ] Ele não repete pergunta já confirmada.
- [ ] Ao lado do chat aparecem o progresso, o que já foi aprendido e o que falta.
- [ ] Você pausa, fecha a página e retoma de onde parou.
- [ ] Com site e perfis informados (e sua autorização), ele lê as páginas públicas e propõe respostas para você confirmar.
- [ ] Clínica de estética e empresa (Laura Hub e Laura Growth) recebem perguntas específicas diferentes.
- [ ] A tela se atualiza ao vivo enquanto a IA trabalha.
- [ ] Se a IA falhar, nada se perde e você pode tentar de novo ou responder à mão.
- [ ] O custo de cada conversa é registrado.

## Etapa 2E — Plano de marketing
**Constrói:** plano gerado a partir do DNA da marca, com versões e aprovação.
**Requisitos:** CAL-02, CAL-04; PRD 2.0 RF-08.

Pronto quando:
- [ ] O plano traz objetivo dos 90 dias, canais escolhidos, pilares com percentual, cadência por rede e os temas dos primeiros 30 dias.
- [ ] Você pede ajustes em texto livre e recebe nova versão.
- [ ] A aprovação registra quem aprovou, quando e qual versão.
- [ ] Qualquer mudança no plano aprovado gera nova versão para aprovação; a anterior continua disponível.
- [ ] Aprovar o plano não aprova nenhum post.

## Etapa 3 — Pauta e texto, com intervenção humana
**Constrói:** o começo do fluxo de criação (etapas 1 e 2 do fluxo: pauta e texto), com os quatro comandos em cada passo: aprovar, pedir ajuste, editar à mão, descartar. O registro de tokens e custo nasce junto, mesmo sem tela ainda.
Cada post nasce com pilar, objetivo e CTA. O agente avança sozinho pelas etapas (piloto automático) e você intervém quando quiser.
O cadastro das APIs de IA, a escolha por etapa e a reserva passaram para a etapa 2A (2026-11-24).
**Requisitos:** POST-01, POST-02, POST-03, POST-04, POST-07, POST-08, POST-09, AGT-01, AGT-03, CST-01.

Pronto quando:
- [ ] Você pede um post ou aceita uma sugestão de pauta.
- [ ] O agente escreve a legenda no tom de voz do kit.
- [ ] Você pede um ajuste com um comentário e recebe nova versão.
- [ ] As versões anteriores ficam guardadas.
- [ ] Cada tarefa do agente grava tokens e custo em dólar.
- [ ] Cada post mostra seu pilar, seu objetivo (atrair, engajar ou converter) e seu CTA.
- [ ] O agente passa da pauta ao texto sem esperar por você, e você consegue intervir no meio.

## Etapa 4 — Imagem e carrossel
**Constrói:** a parte visual do fluxo, usando o kit de marca; envio de material avulso para um post.
**Requisitos:** POST-05, POST-06, MKT-02, CST-02.

Pronto quando:
- [ ] O agente gera uma imagem única e um carrossel com a identidade da marca.
- [ ] Você pede nova versão ou envia sua própria imagem.
- [ ] O post é adaptado ao formato de cada rede.
- [ ] O custo da imagem é registrado em dólar.

**A decidir nesta etapa:** arte criada do zero pela IA ou preenchimento de modelos visuais prontos. Recomendação: começar por modelos prontos, que mantêm a marca consistente e custam menos; medir e depois avaliar a criação livre.

## Etapa 5 — Revisão, aprovação final e entrega manual
**Constrói:** conferência automática (marca, ortografia, formato), prévia de como o post fica em cada rede, aprovação final obrigatória, fila para aprovar vários posts em sequência e o botão para baixar o post pronto (imagens e texto).
**Requisitos:** APR-01, APR-02, APR-04, PUB-03.

Pronto quando:
- [ ] Nenhum post chega ao estado "aprovado" sem a sua aprovação.
- [ ] Você vê a prévia por rede antes de aprovar.
- [ ] Você baixa o post aprovado e publica à mão.
- [ ] Com vários posts aguardando, você aprova ou devolve um após o outro na mesma tela.
- [ ] Qualquer mudança em um post aprovado o devolve para aprovação.
- [ ] Post que contraria o "o que evitar" da marca fica **bloqueado**; regras do setor só geram **alerta**.

**Primeiro marco:** a partir daqui você já economiza tempo na criação dos posts das duas marcas.

## Etapa 6 — Calendário e cadência
**Constrói:** calendário por organização com os estados dos posts; cadência por rede; aviso quando o calendário está abaixo do planejado; plano semanal em lote proposto pelo agente.
**Requisitos:** CAL-01, CAL-02, CAL-03, CAL-04, APR-03.

Pronto quando:
- [ ] Você vê a semana e o mês de cada organização.
- [ ] Cada post mostra seu estado.
- [ ] A plataforma avisa quando faltam posts para cumprir a cadência.
- [ ] Post sem aprovação até a data fica marcado como atrasado e não sai.
- [ ] O agente propõe os posts da semana de uma vez, seguindo cadência e pilares, e você aprova o plano em uma tela.
- [ ] Aprovar o plano cria os posts, mas nenhum fica aprovado para publicação sem a sua aprovação final.

## Etapa 7 — Painel de tokens e custos
**Constrói:** a tela da visão Operação, usando os registros gravados desde a etapa 3.
**Requisitos:** CST-03 a CST-07 ([[PRD - Painel de Custos e Créditos]]).

Pronto quando:
- [ ] Você vê tokens e dólares por organização, post, atividade, agente, formato e período.
- [ ] Você vê o custo médio por post, separado entre publicados e descartados.
- [ ] Você exporta os dados em planilha.

## Etapa 8 — Publicação automática
**Constrói:** conexão das contas e publicação agendada. Uma rede por vez, na ordem em que as autorizações saírem.
**Requisitos:** ORG-06, CNX-01 a CNX-04, PUB-01, PUB-02.

Pronto quando, para cada rede (LinkedIn, Facebook, Instagram):
- [ ] A conta da organização está conectada e mostra estado "ativa".
- [ ] Um post aprovado é publicado sozinho na data e hora marcadas.
- [ ] Uma falha de publicação gera aviso e permite tentar de novo.

Enquanto uma rede não estiver liberada, ela continua na entrega manual da etapa 5.

## Etapa 9 — Período de teste
**Faz:** usar a plataforma de verdade para as duas marcas, sem construir nada novo, anotando problemas.
Duração: a definir.

Pronto quando:
- [ ] A cadência foi cumprida no período.
- [ ] As quatro perguntas do [[PRD - Painel de Custos e Créditos]] têm resposta para imagem e carrossel.
- [ ] Os indicadores do [[PRD - Indicadores de Sucesso]] foram anotados.
- [ ] Existe uma lista de ajustes antes de iniciar a fase 1B.

---

## O que não construir agora
Vídeo, TikTok, anúncios, relatórios, medição de resultados por post, link rastreável, alertas de regras do setor, créditos para o cliente, inglês. Tudo isso tem fase própria em [[PRD - Fases e Escopo]].

## Dependências entre as etapas
- A etapa 0 (autorizações) corre em paralelo e só é cobrada na etapa 8.
- As etapas 6 e 7 podem trocar de ordem entre si.
- As demais seguem a ordem numérica.

Riscos: [[PRD - Riscos e Pontos a Definir]].
