---
tipo: projeto
status: em validação
area: Produto e Agentes de IA
criado: 2026-11-27
---
# Relatório — Ajustes 2A-1 (cadastro de IA)

Ajustes pedidos pelo Paulo depois do primeiro teste no stage, em 26/11/2026. Especificação: [[Especificação - Ajustes 2A-1]]. Branch `ajustes-2a-1-ia`.

## O que mudou
| Ajuste | Antes | Agora |
|---|---|---|
| J1. Teste de imagem | O `chatgpt-image-latest` falhava com o erro 400, porque a OpenAI recusa o campo `response_format` | O campo vai só para os modelos `dall-e-*` |
| J2. Mensagem de erro | Um teste que falhava mostrava "Algo deu errado. Tente de novo" | A tela mostra o motivo guardado no registro, por exemplo "O fornecedor retornou um erro (400)." |
| J3. Modelo reserva | Era obrigatório | É opcional. A atividade fica pronta só com o principal |
| J4. Excluir fornecedor | Não existia | Botão "Excluir" com confirmação. Se o fornecedor estiver em uso numa atividade, a exclusão é bloqueada com um aviso que diz em qual. O histórico de custos é mantido |

**Causa do J2:** o stage passa pela Cloudflare, que pode trocar as respostas com o código 502 pela página de erro dela. O servidor usava esse código para "o fornecedor recusou" e agora usa o 422.

## Banco de dados
- Migração `0006_exclusao_fornecedor.sql`, que só acrescenta: os registros de uso aceitam fornecedor e modelo vazios depois de uma exclusão, e o nome fica guardado.
- O desfazer está num comentário da própria migração.

## Testes
| Verificação | Resultado |
|---|---|
| Testes do servidor (Go) | todos passando, cobertura de 72,5% |
| Verificação e formatação do código (go vet e gofmt) | limpas |
| Checagem de tipos das telas | limpa |
| Testes de tela (Playwright) | 10 de 10 |

## Revisões
- **GERAL e SEGURANÇA**, em claude-sonnet-5-5 e com a skill `code-review-lg` carregada.
- **Rodada 1:** nenhuma pendência que bloqueasse. A GERAL apontou uma pendência importante: a primeira correção do J2 era um remendo na tela. Vieram ainda 4 pendências menores.
- **Rodada 2:** as duas revisões aprovaram, sem nenhuma pendência.

## Limpeza no stage (26/11/2026, feita antes, com cópia do banco)
Foram removidas as 2 contas de teste e os 2 fornecedores "Simulado E2E", com seus modelos, preços e registros simulados.

## Depois da aprovação
Instalar no stage: fazer a cópia do banco, aplicar a migração 0006, guardar o servidor atual como `.bak`, montar e reiniciar.

## Relacionado
[[Especificação - Ajustes 2A-1]] · [[Especificação - Etapa 2A-1]] · [[Roteiro de Construção - Fase 1A]]
