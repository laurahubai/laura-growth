---
name: refinar-prd
description: Aplica decisões de negócio ao PRD do Laura Growth, mantendo PRD, notas de decisão e índices sincronizados. Use ao refinar ou atualizar o PRD.
version: 0.2.0
---
# Refinar PRD

## Quando usar
Sempre que houver decisões novas (nota de decisão, conversa, revisão) que precisem entrar no PRD.

## Entradas
- PRD vigente: docs/prd/prd-lauragrowth20.md
- Nota(s) de decisão indicadas pelo usuário
- PRD - Decisões de Negócio (somente leitura, salvo pedido explícito)

## Passos
1. Leia o PRD e as notas de decisão. Não altere nada ainda.
2. Monte uma tabela: decisão | seção do PRD afetada | texto atual | texto proposto.
3. Liste conflitos e decisões ambíguas como perguntas. Não invente respostas.
4. Apresente a tabela e as perguntas. **Pare e aguarde aprovação.**
5. Após aprovação: aplique as mudanças, suba a versão no cabeçalho e acrescente uma entrada em "Histórico de alterações" (data, resumo, nota de origem).
6. Mude o status da nota de decisão para `aprovado` e escreva no corpo
   "Incorporada ao PRD <versão>". Status válidos no cofre:
   conceito, rascunho, em validação, aprovado.
6b. Garanta que o PRD tenha as propriedades do cofre (tipo, status).
    Ao criar versão nova do PRD, marque a anterior como substituída
    e aponte para a nova.
  
## Regras
- Nunca alterar código.
- Nunca fazer commit sem pedido explícito.
- Todo requisito funcional novo precisa ter critério de "pronto quando".
- Pendências não resolvidas ficam numa seção "Pendências" do PRD, com dono e data, se informados.

## Verificação
- Toda decisão da nota aparece no PRD ou está listada como pendência.
- Nenhuma seção foi removida sem estar na tabela aprovada.