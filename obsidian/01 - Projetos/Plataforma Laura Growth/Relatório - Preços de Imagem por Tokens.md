---
tipo: projeto
status: em validação
area: Produto e Agentes de IA
criado: 2026-11-27
---
# Relatório — Preços de imagem por tokens

Ajuste pedido pelo Paulo em 27/11/2026, depois de testar o `chatgpt-image-latest` no stage. Especificação: [[Especificação - Preços de Imagem por Tokens]]. Branch `ajustes-2a-1-precos-imagem`.

## O que mudou
- **Três preços por tokens.** Os modelos de imagem passam a ter estes preços, em US$ por 1 milhão de tokens e com data de vigência:
  - "Entrada de texto";
  - "Entrada de imagem";
  - "Saída de imagem".
- **Cálculo do custo.** O custo de cada uso é a soma de:
  - tokens de texto × preço de entrada de texto;
  - tokens de imagem enviados × preço de entrada de imagem;
  - tokens gerados × preço de saída de imagem.
- **"US$ por imagem" continua,** só para modelos que não informam tokens, como os `dall-e`. Se houver os três preços por tokens, valem os tokens.
- **Proteções:**
  - **Sem uso informado:** se o fornecedor não disser quantos tokens usou, vale o preço por imagem, se houver; senão, o teste falha com mensagem clara. Nunca fica registrado custo zero.
  - **Contas que não fecham:** se as parcelas de tokens enviadas pelo fornecedor não somarem o total, o pedido é recusado.
- **Histórico.** Os registros antigos não foram alterados. Cada registro mostra como foi calculado: "por tokens" ou "por imagem (preço antigo)".
- **Testes sem custo.** O fornecedor simulado também informa tokens nas imagens, para os testes rodarem sem gastar dinheiro.

## Banco de dados
- Migração `0007_precos_imagem_tokens.sql`, que só acrescenta:
  - um novo tipo de preço, entrada de imagem;
  - duas colunas opcionais nos registros de uso.
- O desfazer está num comentário da própria migração.

## Testes (conferidos pelo orquestrador)
| Verificação | Resultado |
|---|---|
| Testes do servidor (Go) | todos passando |
| Verificação e formatação do código (go vet e gofmt) | limpas |
| Checagem de tipos das telas | limpa |
| Testes de tela (Playwright) | 11 de 11 |

## Revisões
- **GERAL e SEGURANÇA**, em claude-sonnet-5-5 e com a skill `code-review-lg` carregada.
- **Rodada 1:** nenhuma pendência que bloqueasse.
  - A SEGURANÇA apontou uma pendência importante: uma imagem sem uso informado sairia com custo zero e escaparia do limite mensal.
  - Vieram ainda 5 pendências menores.
- **Rodada 2:** as duas revisões aprovaram, sem nenhuma pendência.

## Depois da aprovação
1. Instalar no stage: fazer a cópia do banco, aplicar a migração 0007, guardar o servidor atual como `.bak`, montar e reiniciar.
2. Paulo cadastra os três preços do `chatgpt-image-latest`, copiados da página de preços da OpenAI.

## Relacionado
[[Especificação - Preços de Imagem por Tokens]] · [[Relatório - Ajustes 2A-1]] · [[Especificação - Etapa 2A-1]]
