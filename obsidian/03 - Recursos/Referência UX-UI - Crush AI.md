---
tipo: referência
status: rascunho
area: produto
criado: 2026-11-15
---
# Referência UX-UI - Crush AI

> **O que é:** leitura do **comportamento e da aparência** do site trycrush.ai, como referência para o site e a plataforma da Laura Growth.
> **UX** = como a pessoa usa e se sente navegando. **UI** = a aparência (cores, letras, botões).
> **Fonte:** página inicial pública, lida em 2026-11-15 no computador e no celular. Valores de cor e tamanho foram medidos no próprio site. Não entramos no produto por dentro.
> **Regra:** usar como **inspiração de padrões**, nunca copiar marca, textos, imagens ou personagens.

Relacionado: [[Benchmark - Crush AI]] (fluxo do produto e encaixe com o PRD) · [[Design System - Plataforma Laura Growth]] · [[Benchmark - Cor de Marca]] · [[Especificação - Frontend Mockado]]

## Resumo em 5 pontos
1. **A demo é o centro do site.** Em vez de vídeo, há uma "tela do produto" funcionando dentro da página, que se apresenta sozinha, etapa por etapa.
2. **A IA fala em primeira pessoa** ("Desde ontem fiz 2 anúncios e gastei $14,10 para 2 vendas"). Parece um colega, não um software.
3. **Visual limpo e claro:** fundo cinza-claro, cartões brancos, uma única fonte (Inter) em peso muito forte nos títulos, e manchas de cor suaves e desfocadas no fundo.
4. **Um "toque de mão"** em pontos-chave: etiquetas inclinadas, coloridas, com emoji, como post-its sobre o texto.
5. **Página de vendas longa e bem ordenada:** problema → solução → para quem → como começar → preço → por que funciona → o que esperar → depoimentos → perguntas → chamada final.

## 1. Estrutura da página (ordem das seções)
| # | Seção | O que faz | Padrão de UX |
|---|---|---|---|
| 1 | **Topo (hero)** | Título grande + etiqueta inclinada "sem desperdiçar milhares em testes" + subtítulo + 3 selos (idioma, moeda, 24/7) + botão + linha de garantia + nota 4,8/5 com fotos | Responde em 5 segundos: o que é, para quem, quanto custa, risco zero |
| 2 | **Faixa de anúncios rolando** | Cartões de anúncios reais passando sozinhos (efeito "esteira") com dados: ativo desde, alcance, dias no ar | Prova visual de volume, sem pedir clique |
| 3 | **Demo interativa** | "Veja como seus anúncios são geridos no piloto automático" — tela do produto com 7 capítulos | Mostrar em vez de explicar (ver seção 3 desta nota) |
| 4 | **Números grandes** | 3 números com legenda pequena | Prova social rápida |
| 5 | **O problema** | 4 dores em cartões com emoji, **desalinhados em zigue-zague** | Sensação de "isso sou eu" |
| 6 | **A solução** | 5 cartões de capacidades, cada um com ícone colorido + frase + benefício em negrito entre parênteses | Lista escaneável |
| 7 | **Para quem** | 4 perfis (fundadores, profissionais de marketing, designers, agências), uma frase cada | Cada visitante se reconhece |
| 8 | **Como começar** | 3 passos numerados (01, 02, 03) em linha do tempo vertical, cada um com um mini-cartão ilustrando | Mostra que é rápido ("em menos de 1 hora") |
| 9 | **Preço** | Fundo lilás, 3 planos lado a lado, "mais popular" destacado em escuro | Comparação fácil |
| 10 | **Por que funciona** | Duas colunas: ❌ por que as pessoas falham × ✅ por que a ferramenta acerta | Contraste direto |
| 11 | **Primeiros 30 dias** | Semana 1 a 4 + "meta realista" | Expectativa honesta, reduz medo |
| 12 | **Depoimentos** | Mosaico de cartões com foto, nome, país, data e foto real de uso | Confiança |
| 13 | **Perguntas frequentes** | Lista que abre e fecha (sanfona) | Responde objeções |
| 14 | **Chamada final** | Emoji 🔥 + "Pronto para crescer?" + botão | Última chance de conversão |

O mesmo botão principal aparece **em todas as seções-chave**. A pessoa nunca precisa rolar para encontrar como começar.

## 2. Menu e navegação
- **Menu fixo e flutuante:** uma "pílula" branca com cantos bem arredondados, solta das bordas (não encosta no topo), que acompanha a rolagem.
- Itens curtos: Solução · Como funciona · Biblioteca de anúncios · Blog · Perguntas · Depoimentos · Afiliados · Entrar.
- No celular vira logo + ícone de menu (☰).
- Link escondido "Pular para o conteúdo", que ajuda quem navega pelo teclado ou leitor de tela.

## 3. A demo interativa (o ponto mais forte)
**Como funciona:**
- Uma janela que imita o produto real, com o selo **"Exemplo"** no canto (deixa claro que os dados são inventados).
- Embaixo, uma **barra de capítulos**: ▶ Encontrar · Criar · Testar · Julgar · Piloto automático · Orgânico (**NOVO**). Ela avança sozinha, como um vídeo, e a pessoa pode clicar em qualquer capítulo.
- Um **cursor de mentira** se move e "clica" nos botões (ex.: "Aprovar"), mostrando a ação sem a pessoa fazer nada.
- No celular, os nomes dos capítulos viram **barrinhas de progresso** (estilo stories do Instagram) e aparece uma legenda curta embaixo ("Toda manhã, um recado do Crush...").
- Há uma descrição em texto de toda a demo para leitores de tela.

**O que cada tela mostra (padrões de interface):**
| Capítulo | Tela | Padrão que vale estudar |
|---|---|---|
| Início | "Bom dia, Lucía" + resumo em 1ª pessoa + 4 cartões de números (cada um com legenda simples: "o que uma venda custou") + faixa "esta semana: 3 testes, 1 vencedor" + selo "Piloto automático ligado" | **Número + explicação em linguagem comum** |
| Encontrar | Biblioteca em grade, cartões selecionáveis com "AINDA NO AR · 63 dias"; barra flutuante "1 selecionado → Gerar · 2 créditos" | **Custo em créditos mostrado no próprio botão** |
| Criar / Testar | "3 anúncios prontos — cada um passou pelo Crítico; nada vai para a Meta até você começar" | **Aviso de segurança junto da ação** |
| Julgar | Lista com relógio "48 h", selos "Vencedor" (verde) e "Pausado", barra de orçamento dividida por anúncio | **Status por cor + motivo escrito** |
| Piloto automático | Selo "Até $50 por dia · Pedir antes", gráfico de linha simples, "Decisões recentes" com etiqueta do dia e da ação | **Histórico de decisões legível** |
| Orgânico | "Um post, quatro lugares" com horário por rede e botões "Aprovar" / "Editar texto" | **Aprovação em um clique** |

Navegação interna do produto: **barra inferior** com 5 ícones (Início · Biblioteca · Anúncios · Meta · Mais), padrão de aplicativo de celular.

## 4. Linguagem visual (UI)
### Cores medidas
| Uso | Cor | Código |
|---|---|---|
| Fundo da página | cinza muito claro | `#F5F5F5` |
| Cartões | branco | `#FFFFFF` |
| Texto principal | quase preto azulado | `#111827` |
| Texto secundário | cinzas | `#6B6B6B` · `#897F7F` |
| Botão principal | grafite | `#343434` com texto branco |
| Contorno do botão principal | degradê azul-lavanda → verde-limão → rosa | `#808DD4` → `#D5F09F` → `#EFCAF3` |
| Etiquetas de destaque | verde-limão | `#D5F09F` (borda `#B4D67E`) |
| Seção de preço / selos | lilás | `#F3E3FF` · `#F3E8FF` · `#D8B4FE` |
| Palavra de destaque no título | roxo / azul | `#9333EA` e azul ("Facebook", "piloto automático") |
| Positivo | verde (ex.: custo por venda) | — |

**Leitura:** a base é neutra (cinza, branco, grafite) e a cor entra só em **pequenas doses**: etiquetas, selos, manchas de fundo e uma palavra por título.

### Tipografia
- **Uma única fonte: Inter.**
- Título principal: 48 px no computador, **peso 900** (o mais grosso).
- Títulos de seção: 30 a 36 px, peso 900.
- Títulos de cartão: 18 a 20 px, peso 700 a 800.
- Texto corrido: 14 a 16 px, cinza.
- Títulos centralizados, sempre curtos, em 1 ou 2 linhas.

### Formas e profundidade
- **Botões em pílula** (cantos totalmente arredondados), altura de cerca de 55 px.
- Cartões com cantos de 12 a 16 px e **sombra suave**.
- Selos e chips: pílula com fundo pastel + emoji.
- Largura máxima do conteúdo: cerca de 1.150 px, bem centralizado.
- Muito espaço entre seções (80 a 96 px).

### Elementos decorativos
- **Manchas de cor desfocadas** (verde-limão e lilás) atrás do conteúdo, dando um ar leve e "tecnológico".
- **Etiquetas inclinadas** tipo post-it sobre títulos ("em menos de 1 hora", "sem desperdiçar milhares").
- **Emoji** como ícone em selos, dores e perguntas.
- Fotos reais de clientes usando o produto nos depoimentos.

## 5. Movimento e microinterações
| Comportamento | Detalhe |
|---|---|
| Faixa de anúncios | rola sozinha sem parar, bem devagar (um ciclo a cada 55 s) |
| Demo | troca de tela com subida e esmaecimento curtos (cerca de 0,2 s) |
| Botões e links | mudança de cor suave em 0,15 a 0,3 s |
| Perguntas frequentes | abrem deslizando (0,7 s) |
| Ponto "ao vivo" | pulsa sem parar (ex.: "AO VIVO", "ligado") |
| Cartões de dores | posição em zigue-zague, como papéis soltos |

O movimento é **curto e funcional**: chama atenção sem atrasar a leitura.

## 6. Táticas de conversão (avaliar com cuidado)
- Preço "a partir de $0,90/dia" logo no topo e garantia de 14 dias repetida várias vezes.
- Planos com preço riscado, % de desconto e "≈ $/dia".
- **Janela que aparece sozinha** durante a navegação: "Ganhe até 40% de desconto — diga o que está travando seus anúncios", com 4 respostas clicáveis (um mini questionário).
- Renovação automática por um valor maior após o período (aviso em letra pequena).

> ⚠️ Os itens acima podem gerar sensação de pressão ou reclamações. Já registrado como "evitar sem antes avaliar" em [[Benchmark - Crush AI]].

## 7. O que levar para a Laura Growth (sugestões, a confirmar)
**Site de vendas**
- **Demo com capítulos** em vez de vídeo, com selo "Exemplo" e cursor automático. Para nós: Pauta → Criação → Revisão → Aprovação → Publicação, com um negócio fictício (ex.: uma clínica inventada).
- **Mesma ordem de seções**, trocando "anúncios" por "posts e contatos".
- **Etiqueta inclinada** como assinatura visual, usando a nossa cor ([[Benchmark - Cor de Marca]]).
- **"Primeiros 30 dias" com meta realista**: combina com o tom honesto da marca.
- Botão principal repetido em todas as seções.

**Plataforma**
- Home com **saudação + resumo em 1ª pessoa** ("Esta semana preparei 3 posts, 1 aguarda você").
- **Cada número com uma legenda em português simples.**
- **Custo em créditos dentro do botão** da ação. Ver [[PRD - Painel de Custos e Créditos]].
- Aviso de segurança junto da ação ("nada é publicado sem seu OK"). Ver [[PRD - Permissões e Aprovações]].
- **Histórico de decisões** com etiqueta de dia e de tipo de ação.
- Barra de ações flutuante quando há itens selecionados.

**Não levar**
- Janela de desconto que aparece sozinha, contagem regressiva e renovação por valor maior.
- Promessas numéricas sem comprovação ("3X", "$100M+"). Nossos números serão "a definir" até termos dados reais.

> Nada aqui é decisão aprovada. Para virar regra, registrar em [[Registro de Decisões]] e no [[Design System - Plataforma Laura Growth]].
