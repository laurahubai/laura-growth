---
tipo: prd
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# PRD — Agentes de IA

Parte do [[PRD - Plataforma Laura Growth]]. Parte dos seis agentes do conceito, descritos em [[Agentes de IA - Visão Geral]]. Todas as funções são **planejadas**.

## Princípio
**IA executa, humano aprova.** Nenhum agente publica, gasta orçamento ou altera contas sem autorização de uma pessoa.

## Quando cada agente entra
| Agente | Papel na plataforma | Fase |
|---|---|---|
| **Strategy Agent** | Sugere pautas e temas a partir do kit de marca e do calendário | 1A |
| **Creative Agent** | Escreve textos e cria imagens, carrosséis e vídeos | 1A (imagem e carrossel) · 1B (vídeo) |
| **Social Agent** | Monta o calendário, adapta o post a cada rede, agenda e publica | 1A |
| **Ads Agent** | Prepara, publica e acompanha anúncios pagos | 3 |
| **Performance Agent** | Analisa indicadores e recomenda melhorias | 3 |
| **Automation Agent** | Jornadas e nutrição de leads | A definir |

Ficha existente: [[Strategy Agent]]. As demais fichas ainda precisam ser criadas com o [[Template - Agente]].

## Requisitos comuns a todos os agentes
| Código | Requisito |
|---|---|
| AGT-01 | Trabalhar sempre dentro de uma única organização, usando só o kit de marca e os dados dela. |
| AGT-02 | Parar e pedir aprovação nos pontos definidos em [[PRD - Permissões e Aprovações]]. |
| AGT-03 | Aceitar comentários de ajuste em linguagem comum e gerar nova versão. |
| AGT-04 | Registrar tokens e custo de cada tarefa. Ver [[PRD - Painel de Custos e Créditos]]. |
| AGT-05 | Deixar claro o que foi feito por IA e o que foi editado por pessoa, no histórico do post. |
| AGT-06 | Quando não conseguir concluir uma tarefa, avisar em vez de entregar algo incompleto. |
| AGT-07 | Cadastrar, em Configurações, as **APIs de IA** da plataforma: nome, tipo (texto ou imagem), endpoint (endereço da API), modelo e chave de API. A chave fica cifrada no servidor e nunca é exibida inteira. Vale para a plataforma toda (Operação Laura), não por organização. |
| AGT-08 | Escolher **qual API cada etapa usa**: onboarding e plano semanal, pauta, texto, visual e revisão. A mesma API pode servir a várias etapas. O registro de custo guarda a API usada (AGT-04). |
| AGT-09 | Escolher, para cada etapa, uma **API de fallback** (reserva), do mesmo tipo e diferente da principal. Ela entra no lugar da principal quando esta falha (fora do ar, limite de uso ou erro). Sem fallback, a etapa para e o agente avisa (AGT-06). O registro de custo guarda a API que de fato atendeu. |

## A definir
- Quais fornecedores de IA serão usados para texto, imagem e vídeo. A plataforma já prevê uma API por etapa (AGT-07 e AGT-08, decidido em 2026-11-15); falta escolher os fornecedores.
- Limite de tentativas de ajuste por post (para controlar custo).
