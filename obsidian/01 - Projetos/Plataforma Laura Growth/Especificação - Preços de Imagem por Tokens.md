---
tipo: projeto
status: aprovado
area: Produto e Agentes de IA
criado: 2026-11-27
---
# Especificação — Preços de imagem por tokens

Adjustment to [[Especificação - Etapa 2A-1]] and [[Especificação - Ajustes 2A-1]], requested by Paulo on 27/11/2026 after testing `chatgpt-image-latest` on stage. Technical text in English; UI text in Portuguese. Branch `ajustes-2a-1-precos-imagem` from `main` (3b6d004). Code in `sistema/`.

## Problem
- Image models accept only "US$ por imagem" (plus `token_entrada`). The code blocks `token_saida` for image models (admin_modelos.go ~line 648).
- OpenAI bills GPT image models in tokens with three prices:
  - text input;
  - image input;
  - image output.
- The stage test of `chatgpt-image-latest` (12 input tokens, 1,258 output tokens) was recorded at US$ 5.00, because Paulo had entered US$ 5 "por imagem".

## Product decisions (Paulo, 27/11/2026)
- Image models get three prices, each in **US$ per 1 million tokens**, with a start date (vigência) like text prices. UI labels:
  - "Entrada de texto";
  - "Entrada de imagem";
  - "Saída de imagem".
- The cost of an image call is:
  - text input tokens × "Entrada de texto";
  - plus image input tokens × "Entrada de imagem";
  - plus output tokens × "Saída de imagem".
  - Image input tokens are 0 unless a reference image is sent; that is fine.
- **"US$ por imagem" stays**, but only as a fallback for models that report no token usage (e.g. `dall-e-*`).
  - If a model has the three token prices in effect, tokens are used.
  - If it has only the per-image price, the per-image price is used.
- **History is not rewritten.** Past records keep their stored cost. The history and test result show the pricing basis of each record: "por tokens" or "por imagem (preço antigo)". The basis can be derived from the stored price columns; no data rewrite.
- No price values are seeded. Paulo enters them from OpenAI's price page. "Testar" still refuses when no price is in effect.

## Changes
| ID | What | Acceptance (each with an automated test) |
|---|---|---|
| P1 | New price unit for image input tokens (e.g. `token_entrada_imagem`). For image models, allow `token_entrada` (labeled "Entrada de texto"), `token_entrada_imagem` and `token_saida` (labeled "Saída de imagem"), plus `imagem` as the fallback. Text models are unchanged (no `token_entrada_imagem`, no `imagem`). | Go: each unit is accepted or rejected per service (400 with a clear message). |
| P2 | OpenAI image connector reads `usage.input_tokens_details.text_tokens` and `image_tokens` and `usage.output_tokens`. If the details are missing, all input tokens count as text. Totals are still validated. | Go with an HTTP test server: usage with details, and usage without details. |
| P3 | Cost calculation and recording. `ia_execucoes` stores image input tokens and the image-input price used (new nullable columns, additive). When all three token prices are in effect, the token formula is used; otherwise, when a per-image price exists, the per-image price is used; otherwise the call is refused as today. "Ready" checks for activities and models follow the same rule. | Go: token formula exact to 8 decimals; fallback to per-image; refusal without prices; the activity "o que falta" message is right. |
| P4 | UI. The prices dialog for image models shows the three token prices (US$ per 1M tokens) and "Por imagem (só para modelos sem tokens)". The model list shows the prices in effect. The test result and history show the tokens broken down and the basis ("por tokens" / "por imagem (preço antigo)"). | Playwright: register the three prices on a simulated image model, run Testar, and see the cost and the "por tokens" basis. |
| P5 | The simulated provider returns token usage for images (text and output, image input 0), so P3 and P4 can be tested without spending money. | Covered by P3 and P4 tests. |

## Data
Migration `0007_precos_imagem_tokens.sql`, additive only:
- extend the `ia_precos.unidade` CHECK with the new unit;
- add `ia_execucoes` nullable columns for image input tokens and the image-input price.

The undo goes in a comment. Existing rows are untouched.

## Out of scope
- Image editing with reference images (the field exists for when it arrives).
- Anthropic (no images).
- Recalculating old records.
- 2D.

## Mandatory tests
- Go tests per P1–P5; existing tests passing.
- `go vet`, gofmt and typecheck clean.
- Full Playwright suite.
- `pg_dump` before migrating the VM database.

## Slice
One slice (P1–P5). Reviews: GERAL and SEGURANÇA, both claude-sonnet-5-5, with the `code-review-lg` skill. Blockers go back to the implementer.

## Relacionado
[[Especificação - Etapa 2A-1]] · [[Especificação - Ajustes 2A-1]] · [[Relatório - Ajustes 2A-1]]
