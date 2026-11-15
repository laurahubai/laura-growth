---
tipo: projeto
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# Design System — Plataforma Laura Growth

> **Design system** = conjunto de regras visuais (cores, letras, espaços, componentes) que deixa todas as telas com a mesma cara.
> **Origem:** copiado e traduzido do guia de interface da Laura Hub (`laura-hub-aesthetic-clinic-mpv/docs/ui-laura-hub.md`).
> **Status:** rascunho. As adaptações para a Laura Growth (cores próprias, menu, telas) estão marcadas como **a confirmar**.

## Para que serve
Base visual do frontend (a parte da plataforma que a pessoa vê e usa) da [[PRD - Plataforma Laura Growth]], usada a partir da etapa 0 do [[Roteiro de Construção - Fase 1A]].

## O que foi copiado e o que não se aplica
| Parte do guia original | Situação |
|---|---|
| Princípios, layout, tokens, tipografia, componentes, IA, acessibilidade, idioma | **Copiado** (abaixo) |
| Telas de clínica (agenda, pacientes, estoque, financeiro, procedimentos) | **Não se aplica** — a Laura Growth terá telas próprias (a definir a partir de [[PRD - Fase 1 - MVP de Posts]]) |
| Referência visual Dentlo (Dribbble) | Mantida só como inspiração de estrutura; nunca copiar marca ou imagens |
| Paleta de cores | Copiada da Laura Hub. **A confirmar** se a Laura Growth terá cor principal própria |

## Princípios
Prioridade, nesta ordem:
1. Clareza operacional
2. Segurança e possibilidade de desfazer
3. Hierarquia da informação
4. Consistência do design system
5. Acessibilidade
6. Funcionar bem em qualquer tamanho de tela
7. Acabamento visual
8. Efeitos decorativos

Quando aparência e usabilidade brigarem, vence a usabilidade.

Toda tela deve responder rápido:
1. Onde estou?
2. O que precisa da minha atenção?
3. Qual a próxima ação segura?
4. O que a IA fez e o que ainda depende de uma pessoa?

## Estrutura da tela
```text
┌──────────────────────────────────────────────────────────────┐
│ Menu     │ Barra superior: busca, atalhos, avisos, organização, usuário │
│ lateral  ├────────────────────────────────────────────────────┤
│          │ Cabeçalho: título, contexto e ações                │
│          │ Abas ou filtros                                    │
│          ├────────────────────────────────────────────────────┤
│          │ Conteúdo: grade / tabela / calendário              │
└──────────────────────────────────────────────────────────────┘
```

### Menu lateral
- Largura aberto: 240–248px; recolhido: 72–80px
- Fundo branco, borda sutil à direita, logo no topo
- Itens agrupados por função; item ativo com fundo coral bem claro (`--laura-brand-soft`) e texto/ícone em `--laura-brand-strong`
- Ícones de 18–20px; altura do item 40–44px; cantos do item ativo 8–10px
- Rodapé com conta, organização ou suporte
- Não mostrar itens que o perfil não pode acessar (ver [[PRD - Permissões e Aprovações]])
- Itens do menu da Laura Growth: **a definir**

### Barra superior
- Altura 64–72px, fundo branco, borda sutil embaixo
- Busca global; ações utilitárias à direita
- Seletor de organização (ver [[PRD - Modelo de Organizações]]), avisos, ajuda, avatar
- Ícones sem texto precisam de nome acessível e dica (tooltip)

### Área de conteúdo
- Fundo cinza bem claro; superfícies brancas com borda de 1px
- Margem interna: 24px no computador, 16px em telas menores
- 20–24px entre seções grandes

### Cabeçalho da página
- Título curto (24–28px, não 48px), descrição opcional
- Ação principal à direita; ações secundárias agrupadas
- Abas logo abaixo do título

### Grade
- 12 colunas no computador, espaço de 16–20px
- Um bloco principal largo + uma coluna auxiliar de "atenção rápida"
- Cartões compactos no topo; no máximo 4–5 indicadores visíveis sem rolar
- Não transformar todo texto pequeno em cartão

## Tamanhos de tela
| Largura | Comportamento |
|---|---|
| 1440px ou mais | Menu aberto, grade completa, busca larga |
| 1024–1439px | Menu recolhível, blocos reorganizados, ações secundárias em menu |
| 768–1023px | Menu vira gaveta, indicadores em 2 colunas, tabelas com rolagem lateral |
| Abaixo de 768px | Uma coluna, janelas viram tela cheia, áreas de toque com 40px+ |

Testar em: 1440×900 · 1280×800 · 1024×768 · 390×844.

## Tokens (valores-base do visual)
> **Token** = nome fixo para um valor visual (ex.: `--laura-primary` em vez de um código de cor solto). Mudar o token muda o sistema todo.

### Cores
```css
:root {
  --laura-bg: #f6f7fb;
  --laura-surface: #ffffff;
  --laura-surface-subtle: #fafbfc;
  --laura-surface-muted: #f1f3f7;
  --laura-text: #182230;
  --laura-text-muted: #667085;
  --laura-text-subtle: #98a2b3;
  --laura-text-inverse: #ffffff;
  --laura-border: #e7eaf0;
  --laura-border-strong: #d7dce5;
  /* Ações principais: grafite (decidido em 15/11/2026) */
  --laura-primary: #1f2430;
  --laura-primary-hover: #2b303b;
  --laura-primary-active: #11151c;
  /* Cor da marca Laura Growth: coral (decidido em 15/11/2026) */
  --laura-brand: #f26b3a;        /* logo, ilustrações, detalhes gráficos — nunca texto */
  --laura-brand-strong: #c2410c; /* texto, links, ícone e item ativo do menu */
  --laura-brand-hover: #a8380b;
  --laura-brand-soft: #fff3ed;   /* fundo do item ativo e de destaques */
  --laura-brand-border: #fdd5c2;
  --laura-ai: #7a5af8;
  --laura-ai-soft: #f2efff;
  --laura-success: #17875b;
  --laura-success-soft: #eaf8f1;
  --laura-warning: #b7791f;
  --laura-warning-soft: #fff7e6;
  --laura-danger: #d92d20;
  --laura-danger-soft: #fff0ee;
  --laura-info: #1570ef;
  --laura-info-soft: #eff8ff;
}
```
**Cor da marca: coral, com botões principais em grafite** (decisão de 15/11/2026, ver [[Benchmark - Cor de Marca]]). A Laura Hub segue em azul; os demais papéis (IA, sucesso, alerta, perigo) não mudam.

Contraste testado (padrão WCAG AA pede 4,5 para texto e 3 para elementos gráficos):
| Token | Sobre branco | Uso permitido |
|---|---|---|
| `--laura-primary` (grafite) | 15,5 | Botões principais com texto branco |
| `--laura-brand-strong` | 5,2 | Texto, links, ícones, botões com texto branco |
| `--laura-brand` | 3,0 | Só logo e elementos gráficos, **nunca texto** |

O coral (matiz 17°) foi posicionado entre o vermelho de perigo (4°) e o âmbar de alerta (36°) para não ser confundido com nenhum dos dois.

### Cantos arredondados
```css
:root {
  --radius-xs: 4px;   /* detalhes pequenos */
  --radius-sm: 8px;   /* campos compactos, itens de menu, botões pequenos */
  --radius-md: 12px;  /* campos, botões, tabelas, cartões */
  --radius-lg: 16px;  /* cartões, gavetas, janelas */
  --radius-xl: 20px;  /* gavetas e janelas */
  --radius-pill: 999px; /* etiquetas, filtros selecionados, avatares */
}
```
Não usar o mesmo arredondamento em tudo; não transformar todo botão em pílula.

### Espaçamento
Escala de 4 em 4: `4, 8, 12, 16, 20, 24, 32, 40, 48, 64`
- Cartão compacto: 16px · cartão padrão: 20–24px
- Entre controles: 8–12px · entre cartões: 16–20px · entre seções: 24–32px

### Sombras
Borda antes de sombra.
```css
--shadow-card: 0 1px 2px rgb(16 24 40 / 0.04);
--shadow-overlay: 0 12px 32px rgb(16 24 40 / 0.14);
```
Sombra forte só em menus suspensos, gavetas e janelas.

## Tipografia
Fonte: **Inter** (decidido em 15/11/2026). Gratuita (licença aberta SIL OFL), com bom suporte a acentos do português e números de largura fixa.

| Uso | Tamanho / altura da linha | Peso |
|---|---|---|
| Título da página | 24–28 / 32–36 | seminegrito |
| Título de seção | 18–20 / 26–28 | seminegrito |
| Título de cartão | 14–16 / 20–24 | médio ou seminegrito |
| Número de indicador | 24–32 / 32–40 | seminegrito |
| Texto de trabalho | 14–16 / 20–24 | normal |
| Rótulo | 12–14 / 16–20 | médio |
| Informação auxiliar | 12–13 / 16–18 | normal |

- Números, horários e valores com largura fixa (`tabular-nums`)
- Só a primeira letra maiúscula em títulos, abas e botões
- Nada importante abaixo de 12px

## Componentes essenciais
| Componente | O que é |
|---|---|
| `AppShell` | Moldura da aplicação: menu, barra superior e conteúdo |
| `SidebarNav` | Menu lateral configurável, com permissões e item ativo |
| `CommandMenu` | Busca global com atalho de teclado |
| `PageHeader` | Título, descrição, abas e ações da página |
| `MetricCard` | Indicador: rótulo, valor, comparação com período, minigráfico opcional |
| `ChartCard` | Gráfico com título, seletor de período, legenda e estados vazio/erro |
| `StatusBadge` | Etiqueta de situação: sempre cor + texto (+ ícone quando ajudar) |
| `DataTable` | Tabela densa: linhas de 44–52px, texto 13–14px, números à direita |
| `FilterBar` | Busca, filtros visíveis, filtros avançados e contagem de resultados |
| `Drawer` | Gaveta lateral (420–560px) para ver detalhes sem sair da lista |
| `ConfirmDialog` | Confirmação de ações irreversíveis, dizendo o objeto e a consequência |
| `Toast` | Aviso curto de sucesso ou erro; oferecer "Desfazer" quando possível |
| `EmptyState` | Tela vazia com explicação e próxima ação real |

Situações das etiquetas na Laura Growth (posts, aprovações etc.): **a definir**. Do original, vale o padrão de IA: sugestão, aguardando aprovação, aprovado, rejeitado, falhou.

## Padrões de IA
- Marcar claramente: conteúdo sugerido pela IA, gerado pela IA, ações automáticas, itens aguardando aprovação, falhas.
- Usar a cor `--laura-ai` (roxo) com moderação; nada de tela toda roxa.
- Controle humano: revisar, editar, aprovar, rejeitar, assumir, desfazer, ver histórico (alinhado a [[PRD - Permissões e Aprovações]]).
- Nunca confirmar sozinho: exclusões, mudanças financeiras, comunicações sensíveis, automações em grande escala.
- Não inventar notas de confiança ou porcentagens sem dado real.
- Linguagem direta: "Sugestão da Laura", "Resposta gerada por IA", "Aguardando sua revisão", "Não foi possível executar", "Editar antes de enviar".
- Evitar "mágica", "superpoder", "100% automático".

## Textos da interface
- Português do Brasil por padrão; inglês na versão internacional (ver [[Comunicação Internacional]])
- Datas `dd/MM/aaaa`; horas em 24h; moeda `R$ 12.563,95`
- Botões com verbo específico ("Aprovar post", "Agendar publicação"); evitar "OK", "Clique aqui", "Sim/Não" ambíguos
- Erros em 3 partes: o que aconteceu, impacto, como seguir
- Dados de exemplo fictícios; nunca dados reais

## Estados e interações
- Todo componente que carrega dados prevê: carregando, sucesso, vazio, erro, sem permissão, dado desatualizado
- Botão fica desativado enquanto a ação roda
- Esqueletos (blocos cinza de carregamento) no tamanho do conteúdo final
- Sem botões mortos, sem links `#`, sem menu de função não pronta sem aviso
- Animações de 120–200ms, discretas; respeitar quem pede menos movimento

## Acessibilidade
Meta: **WCAG 2.2 AA** (padrão internacional de acessibilidade).
- Navegação completa pelo teclado, com foco visível
- Campos sempre com rótulo; erros ligados ao campo
- Situação nunca indicada só por cor
- Contraste adequado; zoom sem quebrar a tela
- Gráficos relevantes com resumo em texto ou tabela

## Privacidade na interface
- Não registrar dados sensíveis no console do navegador
- Não guardar senhas/tokens de acesso no navegador sem arquitetura aprovada
- Esconder botão não substitui controle de acesso no servidor
- Ver [[LGPD e GDPR]]

## Gráficos
- Linhas finas, grade sutil, fundo branco, coral (`--laura-brand-strong`) na série principal, no máximo 1–2 cores extras
- Sem 3D, sem degradês fortes, sem arco-íris
- Escolha: linha (evolução), barras (comparação), tabela (precisão)
- Nunca inventar tendência; sempre mostrar o período comparado

## Proibido
Vidro fosco sobre dados · degradê roxo na tela toda · sombras fortes em todo cartão · cantos enormes em controles pequenos · todo botão em pílula · menu só com ícones no computador · cinco estilos de cartão · cores fora dos tokens · gráfico decorativo · cartão dentro de cartão sem motivo · ação crítica só no passar do mouse · cinza claro demais em texto · cópia da marca Dentlo.

## Pronto quando (critérios de aceite)
- [ ] Menu, barra, cabeçalho e conteúdo formam uma moldura coerente
- [ ] Cores, cantos, espaços e letras vêm dos tokens
- [ ] Botões e filtros exibidos funcionam de verdade
- [ ] Estados de carregando, vazio e erro tratados
- [ ] Conteúdo de IA identificado e com revisão humana
- [ ] Testado em computador, notebook, tablet e celular
- [ ] Funciona pelo teclado, com foco visível e rótulos
- [ ] Sem erros no console; verificações automáticas e build passando

## Pontos a confirmar
1. ~~Cor principal~~ — **decidido:** coral + botões grafite ([[Benchmark - Cor de Marca]])
2. ~~Fonte~~ — **decidido:** Inter
3. Itens do menu lateral
4. Situações (etiquetas) dos posts e aprovações
5. ~~Tecnologias do frontend~~ — **decidido:** mesmas da Laura Hub ([[Tecnologias de Base]])

## Relacionado
[[PRD - Plataforma Laura Growth]] · [[Laura Hub x Laura Growth]] · [[Índice - Projetos]]
