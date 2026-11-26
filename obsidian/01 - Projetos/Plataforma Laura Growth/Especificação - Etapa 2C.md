---
tipo: projeto
status: aprovado
area: Produto e Agentes de IA
criado: 2026-11-26
---
# Especificação — Etapa 2C (DNA da marca)

Build spec for stage 2C of [[Roteiro de Construção - Fase 1A]], from MKT-01 and MKT-04 ([[PRD - Fase 1 - MVP de Posts]]) and [[prd-lauragrowth20|PRD 2.0]] RF-02, RF-03 and RF-10. Technical text in English; UI text in Portuguese.

Branch: `etapa-2c-dna-marca`, from `etapa-2b-acessos` after the 2B adjustment (neither 2A-1 nor 2B is in `main` yet; Paulo's decision). Code in `sistema/`.

## Product decisions (Paulo, 26/11/2026)
- Channels: a checklist (Instagram, Facebook, LinkedIn, TikTok, YouTube, Google Meu Negócio, Site/blog, WhatsApp, E-mail) plus an "Outro" text field.
- Only the Roteiro "pronto quando" is in scope. Tone sliders (RF-03) and restoring an old version are left out.
- Alçadas from 2B: viewing the kit, versions and origins requires `ver_marca_posts`; editing and confirming requires `cmo_dna`.

## Goal
The brand kit gains the essential PRD 2.0 fields. "O que evitar" is shown prominently. Each field shows whether Paulo confirmed it or the AI suggested it. Every save creates a version showing who changed what and when. Existing kit data is preserved.

## New fields (added to `kits_marca`)
| Field | Type | Source |
|---|---|---|
| `produtos` | jsonb list `{nome, prioritario}`, max 20, at least one prioritário when the list is not empty (UI warning only) | BD05, BD06 |
| `diferencial` | text | BD12 |
| `modelo_negocio` | `b2b`/`b2c`/`b2b2c`/`outro`/empty, plus `modelo_outro` text | BD07 |
| `regioes` | text (regions and languages) | BD08 |
| `como_compram` | text | BD11 |
| `maior_desafio` | text | BD13 |
| `objetivo_90_dias` | text | Roteiro |
| `canais` | text[] from the fixed codes, plus `canais_outro` text | Roteiro |
| `restricoes_setor` | text (sector restrictions given by the client) | Roteiro |

"O que evitar" = existing `nunca_dizer` + `restricoes_setor`, shown in a highlighted block at the top of the kit page and in the read view. Text limits match the existing kit fields.

## Origin per field (RF-10 subset)
- Table `kit_origens (organizacao_id, campo, origem CHECK IN ('confirmado','sugerido_ia'), definido_por NULL, definido_em)`, PK (org, campo).
- A human save marks every **changed** field `confirmado`. Unchanged `sugerido_ia` fields stay as they are until the user clicks "Confirmar" on that field (also a save, also versioned).
- AI suggestions are written only by a server-side function `gravarSugestoesIA(org, campos)` for stage 2D to use. There is no public route to mark a field as suggested. Tests use that function directly, plus an E2E fixture in `e2e_apoio.go`.
- Migration: every existing non-empty field gets `confirmado`, with `definido_por` set to the kit's `atualizado_por`.

## Versions
- Table `kit_versoes (id, organizacao_id, numero, dados jsonb, origens jsonb, campos_alterados text[], autor_id NULL, autor_tipo CHECK IN ('pessoa','ia'), criado_em)`, unique (org, numero). Append-only.
- Every successful save creates exactly one version: text/list changes, confirmations, AI suggestions, and logo/reference file add or remove (file changes appear in `campos_alterados` as `logo`/`arquivos` with the file name; file bytes are not copied).
- Migration creates version 1 for each existing kit (`autor_tipo='pessoa'`, `autor_id=atualizado_por`).
- Concurrency: a save sends the version number it was based on; if the kit has moved on, the server returns 409 "O kit foi alterado por outra pessoa. Recarregue para ver a versão atual."

## Acceptance criteria (each with an automated test)
| AC | What the test proves |
|---|---|
| C1 | All new fields save and read back; invalid `modelo_negocio`/channel codes and oversize values are rejected (400) |
| C2 | The "O que evitar" block (nunca dizer + restrições do setor) appears at the top of the kit page |
| C3 | Every field shows a badge "Confirmado por você" / "Sugerido pela IA"; a human edit or "Confirmar" turns a suggestion into confirmed; the origin is stored with who and when |
| C4 | Each save creates one version; the history screen lists versions (number, who, when, changed fields) and opens any old version read-only |
| C5 | Stale save → 409 with clear message, and nothing is written |
| C6 | Existing kit data and files are intact after migration 0005, and version 1 exists |
| C7 | Without `ver_marca_posts` → 403 on kit, versions and origins; without `cmo_dna` → 403 on save/confirm; another org's versions → 404 (AC18) |

## Routes (Go)
- `GET /api/kit` adds the new fields, `origens` and `versao`.
- `PUT /api/kit` accepts the new fields and `versao_base`.
- `POST /api/kit/confirmar` `{campos:[...], versao_base}`.
- `GET /api/kit/versoes` (paged, 20 per page), `GET /api/kit/versoes/{numero}`.

## Screens
The kit page gains the "O que evitar" block at the top, the new fields grouped as "Negócio" and "Estratégia", a badge plus "Confirmar" on suggested fields, and a "Versões" link. A new versions page lists versions and shows a read-only view of each.

## Mandatory tests (PRD §15.1)
Go tests per AC (test DB, migration on a copy holding pre-existing kit data for C6); Playwright: fill the new fields → version 2 appears; suggested badge → Confirmar; person without `cmo_dna` sees no edit button and gets blocked by direct request.

## Slices
| Slice | Content | Depends on |
|---|---|---|
| A1 (2B adjustment) | B11: platform admin manages people of any organization (backend + Administração → Organizações screen + tests), on `etapa-2b-acessos` | — |
| C-S1 | Migration 0005, fields, origins, versions, routes, `gravarSugestoesIA`, Go tests C1, C3–C7 | A1 merged (the branch is created after it) |
| C-S2 | Kit screen changes, versions page, E2E C2–C4, C7 | C-S1 contract; runs in parallel with C-S1 using the contract above |

Each slice: GERAL + SEGURANÇA review (both claude-sonnet-5-5), blockers back to the implementer.

## Out of scope
Tone sliders, restoring versions, AI drafting of the kit (2D), the remaining RF-02/RF-03 questions, and conflict states other than confirmed/suggested (disputed, stale, rejected).

## Relacionado
[[Roteiro de Construção - Fase 1A]] · [[prd-lauragrowth20]] · [[Especificação - Etapa 2B]] · [[PRD - Fase 1 - MVP de Posts]]
