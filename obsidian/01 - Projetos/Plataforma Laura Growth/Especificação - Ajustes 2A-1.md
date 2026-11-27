---
tipo: projeto
status: aprovado
area: Produto e Agentes de IA
criado: 2026-11-26
---
# Especificação — Ajustes 2A-1 (cadastro de IA)

Adjustments to [[Especificação - Etapa 2A-1]] requested by Paulo after the first stage test on 26/11/2026. Technical text in English; UI text in Portuguese. Branch `ajustes-2a-1-ia` from `main` (8da77c4). Code in `sistema/`.

## Product decisions (Paulo, 26/11/2026)
- The reserve model is **optional** for every activity. An activity is "pronta" with only a working principal. This changes AC36 of the 2A-1 spec (principal **and** reserve) to principal required, reserve optional.
- A platform admin can **delete** an AI provider.
  - If any activity uses one of its models (principal or reserve), deletion is blocked with a message naming the activities.
  - On deletion, the provider, its models, its prices and its encrypted key are removed.
  - The usage history (`ia_execucoes`: tokens, US$, provider name and model id snapshots) is **kept**.

## Fixes
| ID | What | Acceptance (each with an automated test) |
|---|---|---|
| J1 | Image test fails with 400 on `chatgpt-image-latest`. OpenAI rejects `response_format` (`unknown_parameter`) on GPT image models. Send `response_format: b64_json` **only** for `dall-e-*` models; all other models get no `response_format`. | Go test with an HTTP test server: the request body for `chatgpt-image-latest` and `gpt-image-1` has no `response_format`; for `dall-e-3` it has `b64_json`. |
| J2 | A failed model or activity test shows "Algo deu errado. Tente de novo" in the UI, while the record holds the real reason. The test result must show the same safe message that is stored in `ia_execucoes.erro` (e.g. "O fornecedor retornou um erro (400)."), never the key or the raw provider body. | Go test: a provider 400 returns the stored message in the response; Playwright or component test: the message is visible. |
| J3 | Reserve optional. `rota.Pronta` does not need a reserve; with no reserve, execution tries only the principal; "o que falta" no longer lists "modelo reserva"; the UI shows the reserve as "(opcional)" and lets you save and test without one. | Go tests: an activity with only a principal is ready and executes (success and failure paths; failure with no reserve returns the principal's error, one attempt recorded). Existing principal+reserve tests still pass. |
| J4 | Delete provider. `DELETE /api/admin/fornecedores/{id}` (platform admin only). Returns 409 `fornecedor_em_uso` "Este fornecedor está em uso nas atividades: X, Y. Troque o modelo antes de excluir." when any route uses its models. Otherwise, in one transaction, deletes prices, models and the provider. Writes an audit entry if 2A-1 has an admin audit log. The UI gets an "Excluir" button on the provider page with a confirmation dialog that names the provider. | Go tests: non-admin → 403; unknown id → 404; in use → 409 and nothing deleted; success → provider, models and prices gone and `ia_execucoes` rows still present with names. Playwright: admin deletes an unused simulated provider and it leaves the list. |

## Data
Migration `0006_exclusao_fornecedor.sql`, additive only:
- `ia_execucoes.fornecedor_id` and `modelo_id` become nullable, and their foreign keys become `ON DELETE SET NULL`.
- The undo goes in a comment.
- History screens must tolerate null IDs, using the stored names.

## Out of scope
- Deleting a single model.
- Logged-in subscription providers (2A-2).
- Any 2D work.

## Mandatory tests
Go tests per J1–J4; existing tests passing; `go vet`, gofmt and typecheck clean; Playwright for J4 and the reserve-optional save; `pg_dump` before migrating the VM database.

## Slice
One slice (J1–J4). Reviews: GERAL and SEGURANÇA, both claude-sonnet-5-5, with the `code-review-lg` skill. Blockers go back to the implementer.

## Relacionado
[[Especificação - Etapa 2A-1]] · [[Roteiro de Construção - Fase 1A]] · [[prd-lauragrowth20]]
