---
tipo: prd
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# PRD — Modelo de Organizações

Parte do [[PRD - Plataforma Laura Growth]]. Esta nota descreve a parte "multi-tenant": uma plataforma, várias organizações, cada uma isolada.

## Regra principal
**A organização é sempre a dona** do seu espaço: dados, contas conectadas, kit de marca, conteúdos e histórico. Terceiros entram como convidados e, quando saem, nada vai embora com eles.

## Estrutura
```
Plataforma Laura Growth
└── Organização (ex.: Laura Growth, Clínica X)
    ├── Kit de marca
    ├── Contas conectadas (redes sociais, anúncios)
    ├── Calendário, posts e campanhas
    ├── Membros internos
    ├── Terceiros convidados (com permissões sob medida)
    └── Consumo (créditos)
```

## Requisitos
| Código | Requisito | Fase |
|---|---|---|
| ORG-01 | Cada organização tem espaço isolado; ninguém vê dados de outra organização. | 1 |
| ORG-02 | A organização tem um **tipo** (primeiro tipo: clínica de estética; a Laura entra como "empresa"). O tipo define quais alertas de regras do setor se aplicam. | 1 |
| ORG-03 | Uma mesma pessoa pode ter acesso a mais de uma organização e trocar entre elas (ex.: você entre Laura Growth e Laura Hub; um freelancer entre duas clínicas). | 1 |
| ORG-04 | A organização pode convidar e remover pessoas a qualquer momento. | 2 |
| ORG-05 | Ao remover um convidado, todo o trabalho dele permanece na organização. | 2 |
| ORG-06 | As contas de redes sociais e de anúncios são conectadas em nome da organização, não do convidado. | 1 |
| ORG-07 | Todo consumo de IA é registrado por organização. Ver [[PRD - Painel de Custos e Créditos]]. | 1 |
| ORG-08 | A organização pode exportar seus dados e pedir a exclusão. Detalhes dependem de avaliação jurídica; ver [[PRD - Conformidade e Regras do Setor]]. | A definir |
| ORG-09 | Uma agência pode ter um painel próprio reunindo as clínicas que atende, sem se tornar dona dos dados delas. | 4 |

## A definir
- Como tratar redes com várias unidades (uma organização com unidades, ou uma organização por unidade).
- Onde os dados ficam armazenados para clientes da Europa.

Permissões e aprovações: [[PRD - Permissões e Aprovações]].
