---
tipo: projeto
status: em validação
area: Produto e Agentes de IA
criado: 2026-11-15
---
# Tecnologias de Base

Parte da etapa 0 do [[Roteiro de Construção - Fase 1A]].

> **Decidido (15/11/2026):** usar as mesmas tecnologias da Laura Hub.
> **Requisito novo:** a tela precisa se atualizar **ao vivo** enquanto a IA trabalha, como no Claude Code (a pessoa vê o progresso, não uma tela parada).
> Levantamento feito no código da Laura Hub (`laura-hub-aesthetic-clinic-mpv`) em 15/11/2026.

## O que a Laura Hub usa (e a Growth vai repetir)
| Parte | Tecnologia | Em uma frase |
|---|---|---|
| Frontend (tela) | Next.js 16 + React 19 + TypeScript | Estrutura padrão de mercado para aplicações web |
| Dados na tela | TanStack Query | Busca e mantém os dados do servidor atualizados na tela |
| Formulários | React Hook Form + Zod | Formulários com validação dos campos |
| Ícones | Lucide | Biblioteca de ícones gratuita |
| Gerenciador de pacotes | pnpm (monorepo) | Instala as bibliotecas; frontend e "contratos" no mesmo repositório |
| Contratos | Pacote `contracts` | Combinado escrito entre tela e servidor sobre o formato dos dados |
| Backend (servidor) | Go | Linguagem do servidor e da API |
| Filas de tarefas | River (sobre o PostgreSQL) | Executa tarefas demoradas em segundo plano (ex.: agente gerando um post) |
| Banco de dados | PostgreSQL | Banco relacional, com dados separados por organização |
| Ponte com IA | Serviço Python (Starlette) | Conversa com os fornecedores de IA (texto, imagem) |
| Login | Próprio, no backend Go | E-mail e senha, sem serviço externo |
| Hospedagem | Docker Compose + observabilidade (Grafana, Prometheus, Tempo) | Mesma forma de subir e monitorar da Hub; servidor: **a definir** |
| Testes | Vitest, Playwright, axe | Testes automáticos, de tela e de acessibilidade |

Observação: a Hub **não usa Tailwind** (os estilos ficam em CSS próprio, `globals.css`). Os tokens do [[Design System - Plataforma Laura Growth]] entram como variáveis nesse CSS.

## A peça nova: atualização ao vivo
A Hub hoje não tem esse recurso. **SSE aprovado pelo usuário em 2026-11-15.**

**Tecnologia escolhida: SSE (Server-Sent Events)**
> **SSE** = canal aberto em que o servidor vai "empurrando" novidades para a tela, sem a pessoa recarregar. É o mesmo tipo de mecanismo que ferramentas de IA usam para mostrar a resposta aparecendo aos poucos.

Por que SSE e não as alternativas:
| Opção | Veredito |
|---|---|
| **SSE** | **Recomendada.** Simples, funciona no navegador sem biblioteca extra, reconecta sozinho e cobre o caso (servidor → tela) |
| WebSocket (canal nos dois sentidos) | Mais complexo; só compensa se a tela precisasse mandar muitas mensagens em tempo real, o que não é o caso |
| Consultar a cada X segundos | Mais simples ainda, mas a tela "pula" em vez de fluir e gasta mais servidor |

### Como funciona, passo a passo
```text
1. Pessoa pede um post na tela
2. Servidor (Go) cria a tarefa na fila (River) e responde na hora: "tarefa criada"
3. Agente roda em segundo plano e, a cada passo, grava um evento:
   "Lendo kit de marca" → "Escrevendo legenda" → texto aparecendo aos poucos → "Pronto para revisão"
4. Cada evento é salvo no banco e avisado ao servidor (PostgreSQL LISTEN/NOTIFY)
5. Servidor empurra o evento para a tela pelo canal SSE
6. A tela mostra o progresso ao vivo
```

### Garantias de experiência
- **Fechou a aba ou caiu a internet?** A tarefa continua rodando. Ao voltar, a tela busca os eventos que perdeu (cada evento tem número) e continua de onde parou.
- **Mais de uma pessoa olhando?** Todos veem o mesmo progresso (ex.: agência e clínica).
- **Cancelar:** botão "Parar" durante a execução.
- **Custo:** cada evento final grava tokens e custo, alimentando o [[PRD - Painel de Custos e Créditos]].
- **Visual:** segue os padrões de IA do [[Design System - Plataforma Laura Growth]] (marcar o que é da IA, estados "trabalhando", "aguardando sua revisão", "não foi possível executar").

### Tipos de evento (rascunho)
| Evento | O que a tela mostra |
|---|---|
| `etapa` | "Lendo kit de marca…", "Gerando imagem…" |
| `texto_parcial` | Texto da legenda aparecendo aos poucos |
| `resultado` | Versão pronta para revisão |
| `aguardando_humano` | Pede aprovação ou ajuste |
| `erro` | O que falhou e como seguir |
| `concluido` | Fim, com tokens e custo |

## Ainda a definir na etapa 0
- [x] Pasta do código da plataforma: máquina de desenvolvimento `gcp-lhac-compozyos-001`, pasta `/lhac/dev/laura-growth` *(decidido em 2026-11-15)*
- [ ] Servidor onde a plataforma vai rodar em uso real *(a definir; os créditos da GCP vencem em 04/12/2026)*
- [ ] Fornecedores de IA para texto e imagem
- [x] Começar **do zero** com as mesmas tecnologias da Hub *(decidido em 2026-11-15)*, reaproveitando só peças pontuais (como login e o `globals.css`, porque o código da Hub tem muita regra de clínica)

## Onde fica o código
- Repositório: `prplaurahub/laura-growth` (GitHub). Clone de trabalho na máquina `gcp-lhac-compozyos-001`, em `/lhac/dev/laura-growth`.
- A máquina **lê** o repositório com a chave dela, mas só **envia** alterações usando a chave do Mac, emprestada durante a conexão. A chave nunca é copiada para a máquina, que é descartável e compartilhada com outros agentes.

## Relacionado
[[Roteiro de Construção - Fase 1A]] · [[Design System - Plataforma Laura Growth]] · [[PRD - Agentes de IA]] · [[Laura Hub x Laura Growth]]
