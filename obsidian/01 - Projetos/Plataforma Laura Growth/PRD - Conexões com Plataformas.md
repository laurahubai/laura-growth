---
tipo: prd
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# PRD — Conexões com Plataformas

Parte do [[PRD - Plataforma Laura Growth]].

## O que precisa ser conectado
| Plataforma | Uso | Fase |
|---|---|---|
| Instagram | Publicar posts e carrosséis; Reels na 1B | 1A |
| Facebook | Publicar na página | 1A |
| LinkedIn | Publicar na página da empresa | 1A |
| TikTok | Publicar vídeos | 1B |
| Meta (anúncios) | Criar e acompanhar campanhas pagas | 3 |
| Google Ads | Criar e acompanhar campanhas pagas | 3 |
| Google Meu Negócio | Posts e dados do perfil local | 3 |

## Requisitos
| Código | Requisito |
|---|---|
| CNX-01 | A conexão é feita pela organização, com a conta dela, e pertence a ela. |
| CNX-02 | Mostrar o estado de cada conexão (ativa, expirada, com erro) e avisar quando precisar reconectar. |
| CNX-03 | A organização pode desconectar qualquer conta a qualquer momento. |
| CNX-04 | A plataforma pede apenas os acessos necessários para a função usada. |
| CNX-05 | Ler de volta os dados que as plataformas informam (alcance, cliques, mensagens, formulários) para os relatórios. *Fase 3.* |
| CNX-06 | Se uma rede não estiver conectada, o post é entregue pronto para publicação manual. |

## Ponto de atenção para o prazo
Para publicar automaticamente, cada plataforma exige que o aplicativo passe por um **processo de autorização** próprio. Prazos e exigências de cada uma: **a verificar** antes de planejar a data da fase 1A.

Por isso o requisito CNX-06 é importante: permite usar o MVP (criação, intervenção, aprovação, medição de custo) mesmo antes de todas as autorizações saírem.

## A definir
- Se a publicação será feita por conexão direta com cada rede ou por um serviço intermediário de publicação.
- Quais dados de resultado cada plataforma realmente disponibiliza.

Riscos: [[PRD - Riscos e Pontos a Definir]].
