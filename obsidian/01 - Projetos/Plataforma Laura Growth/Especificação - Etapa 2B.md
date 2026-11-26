---
tipo: projeto
status: aprovado
area: Produto e Agentes de IA
criado: 2026-11-25
---
# Especificação — Etapa 2B (Acessos por pessoa e por alçada)

Build spec for stage 2B of [[Roteiro de Construção - Fase 1A]], from [[PRD - Permissões e Aprovações]] (PRM-01 to PRM-05) and [[prd-lauragrowth20|PRD 2.0]] §3 and AC18. Technical text in English (decision of 25/11/2026); UI text in Portuguese. All items formerly marked "a confirmar" were approved by Paulo on 26/11/2026.

Branch: `etapa-2b-acessos`, from `etapa-2a-1-cadastro-ia` (4361f2c), because 2A-1 is not yet in `main`. Code in `sistema/`.

## Product decisions (Paulo, 25/11/2026)
- Alçada list = the Roteiro list (8 items, below).
- Invitations are emailed through the shared Zoho Mail adapter (`127.0.0.1:8030`, also used by Laura Hub).
- The organization owner (`dono`) implicitly holds every alçada; nobody can remove them.
- Ready-made sets (PRM-05) are in scope.

## Goal
A person with the "convidar pessoas e dar acessos" alçada invites people by email to the active organization and toggles each alçada per person, with immediate effect and an audit trail. The server refuses any action outside the person's alçadas, including forged URLs and IDs.

## Alçadas (stable codes; labels in PT-BR)
| Code | Label |
|---|---|
| `ver_marca_posts` | Ver a marca e os posts |
| `cmo_dna` | Conversar com o CMO e editar o DNA da marca |
| `aprovar_plano` | Aprovar o plano de marketing |
| `editar_posts` | Criar e editar posts |
| `aprovar_posts` | Aprovar posts para publicação |
| `conectar_redes` | Conectar e desconectar redes |
| `ver_creditos` | Ver consumo em créditos (US$ stays Administration-only, PRD §11) |
| `gerenciar_pessoas` | Convidar pessoas e dar acessos |

Applied to routes that already exist: reading the brand kit requires `ver_marca_posts`; editing the brand kit requires `cmo_dna`. Viewing the organization header stays open to any member. The other alçadas have no routes yet; later stages call the same check.

Ready-made sets, fixed in code. Applying one only pre-fills the checkboxes, and each box can then be adjusted:
- Somente leitura: `ver_marca_posts`, `ver_creditos`
- Criador de conteúdo: `ver_marca_posts`, `cmo_dna`, `editar_posts`
- Aprovador: `ver_marca_posts`, `aprovar_plano`, `aprovar_posts`
- Gestor de marketing: all except `gerenciar_pessoas`

## Acceptance criteria (each with an automated test)
| AC | What the test proves |
|---|---|
| B1 | A person with `gerenciar_pessoas` invites an email to the active org; Zoho adapter receives one send with the link; without the alçada → 403 |
| B2 | Invite link: single use, expires in 7 days, token stored only as hash; accepting with an existing account requires logging in as the invited email; a new account is created with name + password; the person joins with exactly the alçadas chosen in the invite |
| B3 | Alçadas toggled one by one; change takes effect on the person's very next request (no cache) |
| B4 | Without the alçada, every protected route returns 403, including direct URL and forged IDs; resources of another org return 404 (AC18) |
| B5 | Owner always passes every check; owner's alçadas cannot be edited, owner cannot be removed; a person cannot change their own alçadas |
| B6 | Every grant/revoke/invite/accept/removal is recorded with who, whom, what and when (PRM-04), visible on the Pessoas screen |
| B7 | Applying a ready-made set fills the boxes; saving stores individual alçadas (no link to the set) |
| B8 | Administration → Equipe: a platform admin grants/revokes the platform-admin flag to members of house orgs (Laura Hub, Laura Growth) only; the last admin cannot be removed; audited |
| B9 | Email adapter down or not configured → invite kept as "não enviado", clear message, "Reenviar" issues a new token and invalidates the old one; "Copiar link" shown only right after creation/resend; key never in responses or logs |
| B10 | Invite abuse limits: max 30 pending invites per org and 20 sends per hour per org → 429 with clear message |

## Data (migration `0004_acessos.sql`, additive only; `pg_dump` before running on the VM)
- `alcadas_membro (organizacao_id, usuario_id, alcada, concedida_por, concedida_em)` PK (org, user, alcada); FK to `membros` ON DELETE CASCADE; CHECK alcada IN the 8 codes.
- `convites (id, organizacao_id, email, alcadas text[], token_hash bytea unique, criado_por, criado_em, expira_em, aceito_em, aceito_por, revogado_em, envio_estado CHECK IN ('enviado','falhou','pendente'))`.
- `registro_acessos (id, organizacao_id NULL for platform scope, ator_id, alvo_usuario_id NULL, alvo_email NULL, acao, detalhe jsonb, em)`; append-only.
- Existing `membros.papel` keeps `dono|membro`; `usuarios.admin_plataforma` already exists.

## Rules
- Single server check `exigirAlcada(ctx, org, alcada)` reads `membros` + `alcadas_membro` per request; `dono` short-circuits to allowed. All org-scoped queries filter by session's active org.
- Invite email lowercased; inviting an existing member → 409; re-inviting a pending email replaces the old invite.
- A person with `gerenciar_pessoas` can only grant alçadas; they cannot touch the owner or themselves (B5).
- Email: new config `LG_EMAIL_ZOHO_URL` (default `http://127.0.0.1:8030`) and `LG_EMAIL_ZOHO_CHAVE` in `sistema/.env` (Paulo fills it; never committed). `POST {url}/v1/email/send`, `Authorization: Bearer`, body `{to, subject, text}`, success = 200 + `{"status":"sent"}`, 45 s timeout, no redirects (same contract as Laura Hub). Link base from new `LG_ENDERECO_PUBLICO`.
- Tests use a fake adapter; no real email in automated tests.

## Routes (Go; session required except invite read/accept)
- `GET /api/pessoas` (gerenciar_pessoas): members + alçadas + pending invites.
- `POST /api/convites`, `POST /api/convites/{id}/reenviar`, `DELETE /api/convites/{id}` (gerenciar_pessoas).
- `PUT /api/pessoas/{usuarioId}/alcadas` (gerenciar_pessoas): full set replace, audited per diff.
- `DELETE /api/pessoas/{usuarioId}` (gerenciar_pessoas).
- `GET /api/pessoas/registro` (gerenciar_pessoas).
- `GET /api/convites/{token}` (public, minimal: org name, email masked), `POST /api/convites/{token}/aceitar` (public; creates account or requires matching session).
- `GET /api/sessao` adds `alcadas` for the active org (UI uses it only to hide menus).
- `GET /api/admin/equipe`, `PUT /api/admin/equipe/{usuarioId}` (platform admin).

## Screens
- Operação → **Pessoas** (menu shown only with `gerenciar_pessoas`): list, invite form with checkboxes + set picker, edit alçadas, remove, pending invites (resend, cancel, status), history.
- `/convite/[token]`: accept page (create account or log in).
- Menus/buttons hidden when the alçada is missing (kit edit button hidden without `cmo_dna`).
- Administração → **Equipe**.

## Mandatory tests (PRD §15.1)
Go tests per AC with test DB; Playwright E2E: invite → accept → limited person blocked by URL; owner flow; Equipe page.

## Slices
| Slice | Content | Depends on |
|---|---|---|
| S1 | Migration 0004, `exigirAlcada`, apply to kit routes, pessoas/alçadas routes, sets, audit, `/api/sessao` alçadas, admin Equipe routes (B3–B8 backend) | — |
| S2 | Invites + Zoho adapter + accept flow + limits (B1, B2, B9, B10 backend) | S1 |
| S3 | Screens: Pessoas, convite page, menu gating, Equipe; E2E for all ACs | S1 (API contract above); runs in parallel with S2 |

Each slice: GERAL and SEGURANÇA review (skill code-review-lg), blockers back to the implementer.

## Out of scope
Configurable approval rules (APV-01 to APV-05), org-defined custom sets, owner transfer, org deletion/billing, password reset.

## Relacionado
[[Roteiro de Construção - Fase 1A]] · [[PRD - Permissões e Aprovações]] · [[prd-lauragrowth20]] · [[Especificação - Etapa 2A-1]]

## Adjustment of 26/11/2026 (Paulo's answers to the open questions) — approved 26/11/2026
1. **Who grants access:** only administrators, either the platform admin or the client's organization admin. The organization admin is the owner plus anyone holding `gerenciar_pessoas`, which is itself granted on the assignment screen. An organization admin may grant any alçada (current behavior kept). **New:** a platform admin can manage people and invites of **any** organization from Administração → **Organizações** (list of organizations → that organization's Pessoas page, reusing the same component). Same rules apply: the owner cannot be edited or removed, and the admin cannot edit their own alçadas. Audit entries carry the platform admin as the actor, flagged `via_admin_plataforma`. Routes: `GET /api/admin/organizacoes`, and `/api/admin/organizacoes/{orgId}/pessoas|convites|...` mirroring the org routes, platform admin only (403 otherwise, 404 for an unknown org).
2. **Invite self-accept risk:** accepted until email verification exists.

| AC | What the test proves |
|---|---|
| B11 | A platform admin who is not a member lists organizations, invites into one, and toggles alçadas; a non-admin gets 403 on every `/api/admin/organizacoes*` route; the owner and self rules still hold; audit shows the platform admin as actor |
