---
tipo: prd
status: rascunho
area: Produto e Agentes de IA
criado: 2026-11-15
---
# PRD — Riscos e Pontos a Definir

Parte do [[PRD - Plataforma Laura Growth]].

## Riscos
| Risco | Efeito | Como reduzir |
|---|---|---|
| Autorização das redes sociais demorar | Atrasa a publicação automática | Verificar exigências cedo; usar a entrega para publicação manual ([[PRD - Conexões com Plataformas]]) |
| Escopo grande para uma pessoa construir | MVP demora a sair | Seguir a divisão 1A / 1B e não antecipar fases |
| Vídeo gerado por IA ser caro ou de qualidade irregular | Custo alto, muito retrabalho | Deixar para a 1B e medir o custo antes de apostar |
| Três mercados com regras diferentes | Exigência jurídica e de idioma triplicada | Validar no Brasil e abrir EUA e Europa na fase 4 |
| Custo de IA variar | Planos de créditos mal calculados | Medir por atividade e guardar o preço usado em cada registro |
| Posts com "cara de IA" | Marca perde qualidade | Kit de marca detalhado e intervenção humana em cada etapa |
| Conteúdo com risco regulatório publicado | Problema para a clínica | Alertas, registro de quem aprovou, validação jurídica das regras |
| Convidado com acesso demais | Erro ou uso indevido | Permissões sob medida e regra padrão conservadora |
| Testar só com a Laura | Não revela as necessidades de uma clínica | Clínica parceira logo após a fase 1 |

## Pontos a definir
### Negócio
- [ ] Valor do crédito, regra de conversão e planos
- [ ] Preço e escopo do serviço humano da Laura
- [ ] Como funcionarão os cenários de agência e de marketplace
- [ ] Qual clínica parceira fará o primeiro teste externo
- [ ] Datas de cada fase

### Produto
- [x] Quais etapas do fluxo de criação são paradas obrigatórias: só a aprovação final (decidido em 2026-11-15)
- [ ] Cadência padrão por rede
- [ ] Canal dos avisos de aprovação pendente
- [ ] Lista final de permissões
- [ ] Como tratar clínicas com várias unidades

### Construção
- [x] Ferramenta de IA usada para construir a plataforma: Claude Code
- [ ] Pasta do código e tecnologias de base (hospedagem, banco de dados, login)
- [ ] Fornecedores de IA para texto, imagem e vídeo
- [ ] Conexão direta com as redes ou serviço intermediário

### Jurídico
- [ ] Regras de publicidade por mercado e tipo de profissional
- [ ] LGPD e GDPR ([[PRD - Conformidade e Regras do Setor]])
- [ ] Disponibilidade da marca ([[Checklist de Validação Pré-Lançamento]])
