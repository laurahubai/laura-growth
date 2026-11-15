---
tipo: roteiro
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# Roteiro de Construção — Fase 1A

Parte do [[PRD - Plataforma Laura Growth]]. Detalha, em ordem, como construir o que está em [[PRD - Fase 1 - MVP de Posts]].

> Roteiro **planejado**. Datas e ferramentas: a definir.

## Como usar este roteiro
- Uma etapa por vez. Só passe para a próxima quando a lista "pronto quando" estiver toda marcada.
- Cada etapa termina em algo que você consegue **ver e testar**.
- Ao pedir a etapa para a ferramenta de IA, copie os requisitos citados (os códigos levam às notas do PRD).
- A partir da **etapa 5** a plataforma já é útil no dia a dia, mesmo sem publicação automática.

## Visão geral
| Etapa | Entrega | Já dá para usar? |
|---|---|---|
| 0 | Preparação | — |
| 1 | Acesso e organizações | Não |
| 2 | Kit de marca | Não |
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
- [ ] Definir em qual pasta ficará o código da plataforma (o cofre `obsidian/` é só para documentação) *(a definir)*.
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
- [ ] Você entra na plataforma com sua conta.
- [ ] Existem duas organizações: Laura Growth e Laura Hub.
- [ ] Você troca de uma para a outra e cada uma mostra só o que é dela.

## Etapa 2 — Kit de marca
**Constrói:** tela para cadastrar e editar o kit de cada organização, incluindo os pilares de conteúdo; envio de arquivos; onboarding assistido, em que o agente rascunha o kit a partir do site e dos perfis da marca.
**Requisitos:** MKT-01, MKT-03, MKT-04, MKT-05.

Pronto quando:
- [ ] Os dois kits estão cadastrados com o material da etapa 0.
- [ ] O kit de uma organização não aparece na outra.
- [ ] Cada kit tem de 3 a 5 pilares de conteúdo.
- [ ] Você informa site e perfis de uma marca, recebe um rascunho do kit e corrige antes de salvar.

O plano do primeiro mês, que fecha o onboarding, só passa a existir na etapa 6.

## Etapa 3 — Pauta e texto, com intervenção humana
**Constrói:** o começo do fluxo de criação (etapas 1 e 2 do fluxo: pauta e texto), com os quatro comandos em cada passo: aprovar, pedir ajuste, editar à mão, descartar. O registro de tokens e custo nasce junto, mesmo sem tela ainda.
Cada post nasce com pilar, objetivo e CTA. O agente avança sozinho pelas etapas (piloto automático) e você intervém quando quiser.
É aqui que a IA de verdade entra: a etapa inclui o cadastro das APIs de IA (chave, endpoint e modelo) e a escolha da API de cada etapa, com uma API de fallback por etapa.
**Requisitos:** POST-01, POST-02, POST-03, POST-04, POST-07, POST-08, POST-09, AGT-01, AGT-03, AGT-07, AGT-08, AGT-09, CST-01.

Pronto quando:
- [ ] Você pede um post ou aceita uma sugestão de pauta.
- [ ] O agente escreve a legenda no tom de voz do kit.
- [ ] Você pede um ajuste com um comentário e recebe nova versão.
- [ ] As versões anteriores ficam guardadas.
- [ ] Cada tarefa do agente grava tokens e custo em dólar.
- [ ] Você cadastra uma API de IA em Configurações e a chave não aparece inteira depois de salva.
- [ ] Você escolhe APIs diferentes para pauta e texto, e o registro de custo mostra qual foi usada.
- [ ] Você define um fallback para uma etapa; quando a principal falha, a tarefa é concluída pela reserva e o custo mostra a reserva.
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
Convites e permissões, vídeo, TikTok, anúncios, relatórios, medição de resultados por post, link rastreável, alertas de regras do setor, créditos para o cliente, inglês. Tudo isso tem fase própria em [[PRD - Fases e Escopo]].

## Dependências entre as etapas
- A etapa 0 (autorizações) corre em paralelo e só é cobrada na etapa 8.
- As etapas 6 e 7 podem trocar de ordem entre si.
- As demais seguem a ordem numérica.

Riscos: [[PRD - Riscos e Pontos a Definir]].
