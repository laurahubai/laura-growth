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

## Pesquisa de integração — Instagram e LinkedIn (24/11/2026)
Pesquisa na internet. A maior parte das fontes é de empresas que vendem serviços de publicação; só o limite do Instagram veio da documentação oficial da Meta. Confirmar na documentação oficial antes de construir.

**API ou MCP?** A plataforma publica pela **API oficial** de cada rede (API é a "porta de entrada" que um sistema usa para falar com outro). MCP é um jeito de dar ferramentas a um assistente de IA; Instagram e LinkedIn **não têm MCP oficial**. Os MCPs existentes (Postiz, Upload-Post, SimplePost, Postproxy) são de terceiros e apenas repassam para as APIs oficiais. Não servem como base da plataforma.

**Instagram (API da Meta)**
- Só funciona com conta **profissional** (Empresa ou Criador).
- Publicar em contas de clientes exige **revisão do aplicativo** e **verificação da empresa** na Meta. Publicar só na própria conta não exige.
- Permissão de publicação: `instagram_business_content_publish`.
- Limite oficial atual: **50 posts a cada 24 horas** por conta; carrossel conta como 1.
- Prazo da revisão: fontes variam de dias a mais de 60 dias — **a verificar**.

**LinkedIn**
- Publicar na **página da empresa** exige a "Community Management API", com **revisão manual** em duas etapas (acesso de desenvolvimento → acesso padrão, que pede vídeo do app e contas de teste).
- Quem conecta precisa ser administrador da página.
- Publicar no perfil pessoal não precisa de aprovação.
- Prazo: fontes variam de 1 semana a meses — **a verificar**.

**Recomendação (a confirmar):** dar entrada nas duas aprovações já; enquanto não saem, usar a entrega para publicação manual (CNX-06). Serviço intermediário (Upload-Post, Ayrshare e similares) só se as aprovações travarem — custo **a definir** e cria dependência de terceiro.

Fontes: [Meta — Content Publishing](https://developers.facebook.com/docs/instagram-platform/content-publishing/) · [Meta — limite de publicação](https://developers.facebook.com/docs/instagram-platform/instagram-graph-api/reference/ig-user/content_publishing_limit) · [CreatorOS — LinkedIn para páginas](https://www.creatoros.ca/blog/linkedin-api-for-posting-to-company-pages/) · [Upload-Post — LinkedIn API](https://www.upload-post.com/linkedin-api/) · [Postproxy — MCPs para redes sociais](https://postproxy.dev/blog/mcp-servers-for-social-media/)

## A definir
- Se a publicação será feita por conexão direta com cada rede ou por um serviço intermediário de publicação.
- Quais dados de resultado cada plataforma realmente disponibiliza.

Riscos: [[PRD - Riscos e Pontos a Definir]].
