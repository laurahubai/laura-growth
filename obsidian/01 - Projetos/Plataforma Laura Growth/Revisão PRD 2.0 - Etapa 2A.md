---
tipo: revisão
status: aprovado
area: Produto e Agentes de IA
criado: 2026-11-24
---
# Revisão PRD 2.0 — Etapa 2A (Cadastro de IA)

Revisão do [[prd-lauragrowth20|PRD LauraGrowth 2.0]] **v1.1**, com foco no que impede construir e fechar a etapa 2A do [[Roteiro de Construção - Fase 1A]]. Critérios da rubrica `prd-rubrica` (testabilidade, ambiguidade, consistência, escopo, dependências, não funcionais, pendências).

> Por regra do projeto, a revisão fica no cofre (e não em `docs/prd/revisoes/`, como pede a rubrica).

> **Achado 2 incorporado ao PRD 1.2; achados 1 e 3 incorporados ao PRD 1.3** em 24/11/2026. **Achados 4 a 11 incorporados ao PRD 1.4** em 25/11/2026 (decisões abaixo).

## Veredito (atualizado em 24/11/2026)
- **2A-1 (chave de API): LIBERAR COM RESSALVAS.** Os achados 4 a 9 são resolvidos durante a construção.
- **2A-2 (assinatura logada): NÃO LIBERAR** até conferir os termos de uso.

Veredito original: NÃO LIBERAR, por causa dos achados 1 a 3, resolvidos pelas decisões abaixo.

**Decisões que destravam (recomendação):**
1. **Quem cadastra:** ~~por organização~~. **Decidido (24/11/2026): cadastro da plataforma toda**, como no AGT-07, por causa da cobrança em créditos (achado 2).
2. **Decidido (24/11/2026). Assinatura logada:** separar a 2A em duas partes. A **2A-1 (chave de API)** começa já; a **2A-2 (assinatura logada)** só depois de conferidos os termos de uso e definido como a plataforma se conecta à assinatura (achado 1).
3. **Decidido (24/11/2026). Teste sem fornecedor contratado:** aceitar um **fornecedor simulado**, identificado como tal (§2, item 8), para os testes automáticos, mais **uma chave real** de texto para o teste final. Os "2 por serviço" com chaves reais passam a ser cobrados quando os fornecedores forem contratados (achado 3).

## Decisões de 25/11/2026 (incorporadas ao PRD 1.4)
- **Achado 4:** serviços texto e imagem; atividades chat do CMO, plano, pauta, texto, imagem e revisão (Guardian); principal e reserva **obrigatórios** (a atividade não roda sem os dois).
- **Achado 5:** limite mensal em US$ e percentual de aviso (padrão 80%) por fornecedor, com exceção opcional por modelo; no limite, passa para a reserva; sem reserva disponível, para e avisa.
- **Achado 6:** cadastro de modelos com "Ler modelos disponíveis" e inclusão manual; endereço preenchido pelo tipo (OpenAI, compatível com OpenAI, Anthropic, simulado) e editável, com proteção contra SSRF; preços manuais por modelo, com vigência.
- **Achado 7:** novos critérios AC32 a AC39, incluindo o botão "Testar" por serviço (texto: "Oi, qual modelo é você?"; imagem: imagem simples).
- **Achado 8:** administrador da plataforma é uma marca na pessoa (só o Paulo na 2A); interfaces separadas de **Administração** e **Operação** no mesmo app.
- **Achado 9:** chave cifrada no banco com segredo derivado de `LG_CHAVE_CIFRAGEM` (gerada pelo Paulo, só no `.env`); cofre dedicado antes do primeiro cliente.
- **Achado 10:** em conflito com AGT-07/08/09 e CST-01, vale o PRD 2.0.
- **Achado 11:** avaliação de fornecedores e LGPD entra nas Pendências (§19).

## Achados

| # | Severidade | Seção | Achado | Sugestão |
|---|---|---|---|---|
| 1 | bloqueante | §8.3; §19; Roteiro 2A ("Antes de começar") | **Assinatura logada sem base para construir.** Os termos de uso não foram conferidos (o próprio roteiro manda conferir antes de começar). O PRD também não diz **como** a plataforma usa a assinatura (por onde entra, como renova o acesso, onde ficam os dados de acesso). Sem a API não há contagem oficial de tokens, então não está definido de onde vêm os tokens que sustentam o "US$ estimado" (AC29). | Separar em 2A-1 (chave) e 2A-2 (assinatura). Na 2A-2, antes de construir: (a) termos conferidos por fornecedor; (b) definir o mecanismo de conexão; (c) definir como contar tokens (informado pela ferramenta ou estimado pelo tamanho do texto, marcando a diferença). **Decidido: divisão aceita, incorporada ao PRD 1.3.** (a), (b) e (c) ficam para antes da 2A-2. |
| 2 | bloqueante | §8.3; AC30; §9 (AIProvider); [[PRD - Agentes de IA]] AGT-07 | **Contradição sobre quem cadastra.** O AGT-07, citado como requisito da 2A, diz que o cadastro "vale para a plataforma toda (Operação Laura), não por organização". O §8.3 e o AC30 dizem que **clientes** cadastram por chave de API, e a entidade AIProvider tem "organização autorizada". Isso muda as tabelas do banco, a tela e quem paga a conta do fornecedor. | **Decidido em 24/11/2026:** vale o AGT-07. O cadastro é **da plataforma toda** (Operação Laura), não por organização, porque a plataforma paga os fornecedores e cobra as organizações em créditos. **Incorporado ao PRD 1.2** (§8.3, §9, AC30). |
| 3 | bloqueante | §19 (1ª pendência); Roteiro 2A ("pronto quando"); AC29 | **Não há fornecedor contratado**, mas o "pronto quando" exige 2 fornecedores por serviço (texto **e** imagem), a reserva concluindo a tarefa quando o principal falha e tokens/US$ gravados. Isso pede pelo menos 4 contas reais. A etapa pode ser construída, mas não fechada. | Aceitar fornecedor simulado nos testes e 1 chave real de texto no teste final; mover "2 por serviço com chaves reais" para quando houver contratação. Registrar a decisão no roteiro. **Decidido e incorporado ao PRD 1.3 e ao roteiro.** |
| 4 | importante | §8.3; Roteiro 2A; AGT-08 e AGT-09 | **"Serviço" e "etapa" não estão definidos no PRD 2.0.** O §8.3 fala em "texto, imagem etc."; o roteiro manda escolher fornecedor "por etapa", mas a lista de etapas só existe no AGT-08 antigo (onboarding e plano semanal, pauta, texto, visual, revisão), que não inclui chat do CMO, plano de marketing e Brand Guardian. Também não diz se "2 por serviço" é regra que o sistema obriga ou só recomendação. | Listar no §8.3 os serviços do MVP (texto e imagem) e as atividades que escolhem fornecedor: chat do CMO, plano de marketing, pauta, texto, imagem, revisão (Guardian). Dizer que o sistema **avisa** quando falta a reserva, sem impedir o uso. |
| 5 | importante | §8.3 ("definir limites"); §9 (AIProvider) | **"Limites" é vago.** Não diz limite de quê (US$, tokens, pedidos), por qual período, nem o que acontece ao atingir. Sem isso o item não é testável. | Proposta: limite mensal em US$ por fornecedor; ao atingir, usa a reserva; sem reserva, para e avisa (AGT-06). |
| 6 | importante | §8.3; §9 (AIProvider, AgentRun); [[PRD - Painel de Custos e Créditos]] CST-02 e CST-07 | **De onde vem o preço para calcular o US$ não está definido.** Falta uma tabela de preços por modelo (entrada e saída), guardada em cada registro (CST-07). Imagem costuma ser cobrada por unidade, não por token (CST-02), e o §8.3 diz só "tokens e US$". A AIProvider também não tem modelo nem endpoint (o AGT-07 tinha). | Acrescentar ao §9: modelo, endpoint e preço por modelo (por token ou por unidade, com data de vigência). O AgentRun guarda o preço usado. Preços: cadastrados à mão (valores a definir). |
| 7 | importante | §15 (critério de saída da 2A = AC29–30); Roteiro 2A | **Critério de saída incompleto.** O roteiro cobra chave mascarada, escolha por etapa e reserva funcionando, mas o §15 só pede AC29 e AC30; não há AC para reserva, máscara da chave ou limites. Além disso, o AC29 fala de "qualquer atividade de IA", e na 2A ainda não existe atividade (o chat é na 2D). | Criar ACs: (a) chave nunca aparece inteira após salva; (b) principal falha e a reserva conclui, com o registro mostrando qual atendeu; (c) limite atingido segue a regra do achado 5. Incluir na 2A um botão **"testar fornecedor"** que faz uma chamada curta e gera o registro de tokens e US$. |
| 8 | importante | §3; §15 (2A antes de 2B); [[PRD - Painel de Custos e Créditos]] | **Dependência invertida com os acessos.** Cadastrar fornecedor e ver US$ são alçadas, mas as alçadas só chegam na 2B. O Painel de Custos diz que a organização **nunca** vê US$ (coerente com a decisão do achado 2). | Na 2A, só o administrador da plataforma cadastra e vê US$. Na 2B, "cadastrar fornecedores de IA" deixa de ser alçada da organização e vira permissão da equipe Laura. |
| 9 | importante | §8.3; §12 (Segurança: "secret vault"); RF-13; AGT-07 ("cifrada no servidor") | **Onde ficam as chaves não está decidido.** O PRD fala em cofre de segredos; o AGT-07, em chave cifrada no servidor; a [[Tecnologias de Base]] não define nenhum dos dois. Vale também para os dados de acesso da assinatura. | Para a 2A: chave cifrada no banco, com a chave de cifragem fora do banco (arquivo de configuração do servidor), nunca em logs, prompts ou na tela. Cofre dedicado fica para antes de entrar o primeiro cliente. |
| 10 | menor | Roteiro 2A ("Requisitos"); §1.3 | **Requisitos espalhados em duas fontes.** A 2A cita AGT-07/08/09 e CST-01 do PRD anterior (só para consulta) junto com o §8.3. Quando divergem (achado 2), não está dito qual vale. | Trazer para o §8.3 o que ainda vale do AGT-07/08/09 e do CST-01 e escrever: "em caso de conflito, vale o PRD 2.0". |
| 11 | menor | §12 (Privacidade/LGPD) | Avaliar fornecedores e transferência internacional antes de mandar dados de clientes. Não bloqueia a 2A (só a casa usa agora). | Exigir antes de cadastrar fornecedor para a primeira organização cliente. |

## O que já está bom para a 2A
- Decisões de negócio claras: 2+ fornecedores, chave ou assinatura, assinatura só para a casa, US$ estimado marcado (§8.3).
- Entidades AIProvider e AgentRun já previstas (§9).
- Regras de segurança gerais (chave fora de prompts, logs e tela) já escritas (RF-13, §12).

Relacionado: [[prd-lauragrowth20|PRD LauraGrowth 2.0]] · [[Roteiro de Construção - Fase 1A]] · [[Adequação ao PRD 2.0]] · [[PRD - Agentes de IA]] · [[PRD - Painel de Custos e Créditos]] · [[Índice - Projetos]] · [[Home]]
