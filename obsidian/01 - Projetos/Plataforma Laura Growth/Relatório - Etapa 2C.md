---
tipo: projeto
status: em validação
area: Produto e Agentes de IA
criado: 2026-11-26
---
# Relatório — Etapa 2C (DNA da marca)

Resumo da construção da etapa 2C do [[Roteiro de Construção - Fase 1A]], conforme a [[Especificação - Etapa 2C]]. Branch: `etapa-2c-dna-marca`, feita a partir da `etapa-2b-acessos` já com o ajuste B11, porque a 2A-1 e a 2B ainda não estão na `main` (decisão sua).

## O que foi feito
- **Kit ampliado:** o kit de marca ganhou os campos do PRD 2.0, em dois grupos:
  - **Negócio:** produtos ou serviços (até 20, marcando os prioritários; aviso se nenhum for prioritário), diferencial, modelo de negócio (B2B, B2C, B2B2C ou outro), regiões e idiomas, e como os clientes compram.
  - **Estratégia:** maior desafio, objetivo dos próximos 90 dias, canais escolhidos (Instagram, Facebook, LinkedIn, TikTok, YouTube, Google Meu Negócio, Site/blog, WhatsApp, E-mail e "Outro") e restrições do setor.
- **"O que evitar" em destaque:** um bloco no topo da página do kit reúne "nunca dizer" e as restrições do setor. Ele mostra o que está salvo e aparece também nas versões antigas.
- **Origem de cada informação:** cada campo mostra "Confirmado por você" ou "Sugerido pela IA". Editar e salvar um campo sugerido, ou clicar em "Confirmar", o torna confirmado. Fica registrado quem confirmou e quando.
  - A IA só poderá sugerir por uma função interna do servidor, que o Chat do CMO (2D) vai usar. Não existe endereço público para isso.
  - A IA **não sobrescreve** um campo que você já confirmou. Ver a pergunta 1 abaixo.
- **Versões:** toda gravação gera uma versão nova, incluindo confirmações e a troca de logo ou de arquivos. A tela **Versões** lista número, quem mudou, quando e o que mudou, e abre qualquer versão antiga só para leitura. A versão antiga guarda só a identificação da pessoa, e o nome é buscado na hora de mostrar (cuidado com dados pessoais, LGPD).
- **Duas pessoas editando ao mesmo tempo:** se o kit foi alterado por outra pessoa depois que você abriu a página, o servidor recusa a gravação com a mensagem "O kit foi alterado por outra pessoa. Recarregue para ver a versão atual." Nada é gravado e o seu texto continua na tela.
- **Acessos (da 2B):** para ver o kit, as versões e as origens é preciso a alçada "Ver a marca e os posts". Para editar e confirmar é preciso "Conversar com o CMO e editar o DNA da marca". O servidor recusa mesmo quando alguém força pelo endereço da página, e dados de outra organização não aparecem.
- **Dados antigos preservados:** a atualização do banco (migração 0005) só acrescenta. Os campos que já estavam preenchidos ficam como "Confirmado por você", e cada kit existente ganha a versão 1.

## Testes rodados (26/11/2026, na branch integrada)
- Servidor (Go): formatação ok, `go vet` sem avisos, `go test` **aprovado**, cobertura de 72,4%.
- Dados antigos: o teste automático cria um kit com arquivos na estrutura antiga e depois aplica a migração. Além dele, uma **cópia** do banco atual (2 kits e 1 arquivo) recebeu as migrações 0004 e 0005. Os dados e os arquivos ficaram idênticos e cada kit ganhou a versão 1. A cópia foi apagada em seguida; o banco do stage não foi tocado.
- Verificação de tipos e montagem das telas: **aprovadas**.
- Testes de ponta a ponta no navegador (Playwright): **10 de 10 aprovados**. Eles cobrem preencher os campos novos, ver a versão 2 e abri-la, "O que evitar" no topo, o fluxo de sugestão da IA até "Confirmado por você", a gravação em conflito e a pessoa sem acesso, além dos testes das etapas anteriores.
- As duas partes (C-S1, servidor; C-S2, telas) passaram por duas revisões cada, uma geral e uma de segurança. Nenhum problema bloqueante ficou aberto. Os pontos importantes foram corrigidos: o nome da pessoa não fica gravado nas versões, a IA não sobrescreve um campo confirmado e o "Confirmar" agora também confere a organização ativa.

## O que ficou de fora e por quê
- **Como combinado na especificação:** controles deslizantes de tom de voz, restaurar uma versão antiga, a IA rascunhar o kit (vem na 2D), as demais perguntas do RF-02/RF-03 e outros estados de origem (contestado, desatualizado, rejeitado).
- **Implantação no stage:** não foi feita. É preciso fazer cópia do banco, aplicar as migrações 0004 e 0005 e trocar o programa do servidor. Fica para quando você quiser testar.
- **Aviso técnico:** a montagem das telas avisa que a versão do Next.js (a biblioteca das telas) tem uma falha de segurança conhecida e pede atualização. A montagem para na primeira tentativa e funciona na segunda. A atualização não fez parte desta etapa; recomendo tratá-la numa tarefa separada.
- **Pontos menores das revisões, sem risco:** o selo diz sempre "Confirmado por você", mesmo quando quem confirmou foi outra pessoa (é o texto da especificação). A troca de página na lista de versões mostra "Carregando…" na tela inteira.

## Perguntas para você (Paulo)
1. **A IA pode trocar o que você já confirmou?** Hoje, por segurança, não: a sugestão da IA ignora campos confirmados. Na 2D, quer que a IA possa propor uma mudança num campo confirmado (por exemplo, como sugestão pendente de aprovação)?

## Relacionado
[[Especificação - Etapa 2C]] · [[Roteiro de Construção - Fase 1A]] · [[Relatório - Etapa 2B]] · [[prd-lauragrowth20]] · [[Índice - Projetos]]
