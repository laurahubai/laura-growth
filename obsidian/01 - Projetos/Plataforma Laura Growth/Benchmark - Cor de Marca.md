---
tipo: projeto
status: aprovado
area: Marca e Posicionamento
criado: 2026-11-15
---
# Benchmark — Cor de Marca

> **Benchmark** = comparação com produtos parecidos para tomar uma decisão melhor.
> **Pergunta:** a Laura Growth deve usar a mesma cor principal da Laura Hub (azul-índigo `#4f5ff5`) ou ter cor própria?
> **Fonte:** telas web do Mobbin (biblioteca de telas de apps reais), pesquisa feita em 15/11/2026 com dois temas: "painel de automação de marketing" e "agendamento de posts em redes sociais".
> **Atenção:** as cores abaixo foram **observadas a olho** nas imagens, não medidas. Servem para ver tendências, não como códigos exatos.

## O que encontramos (20 produtos)
| Família de cor | Produtos | Onde aparece |
|---|---|---|
| **Neutro (preto/grafite) nos botões** | [Square](https://mobbin.com/screens/016da651-284b-4062-9938-e04e1122d18a), [Klaviyo](https://mobbin.com/screens/9fd76029-a965-453f-8804-f2f4745c6fb2), [Shopify](https://mobbin.com/screens/04f1c916-c9b9-45bf-8aff-83ca0a27f0b6), [Customer.io](https://mobbin.com/screens/2fdd46ae-e901-4277-b3f5-aa6a4c918a35), [Cofounder](https://mobbin.com/screens/87d0c6f8-0dd0-4bb9-83f5-d7a743b4a82e), [Threads](https://mobbin.com/screens/a213e175-dcb8-4d1b-94dd-475af771a9f0) | Botão principal escuro; a cor da marca fica só no logo e em detalhes |
| **Azul / índigo** | [Calendly](https://mobbin.com/screens/c83cef3b-705a-43d2-bd2d-8d40c0643b41), [Canny](https://mobbin.com/screens/eb833b3a-040e-4a7f-8900-5e3ad33b3a9a), [Featurebase](https://mobbin.com/screens/24db4970-9824-41ef-8dc3-b522dd50ef4a), [Semrush](https://mobbin.com/screens/57e3ccd6-4cc8-437a-9c37-8b1d2002e8d9), [Basecamp](https://mobbin.com/screens/25c42c82-898e-423e-83aa-24f51379f3b9), [Juicebox](https://mobbin.com/screens/b324dde2-8cf1-4230-9be3-4cb75ed28d6a) (gráficos) | Botões e item ativo do menu. Canny e Featurebase usam um índigo muito parecido com o da Laura Hub |
| **Roxo** | [Canva](https://mobbin.com/screens/40ee9555-0597-43fb-93f9-92bc4f82eb83), [Juicebox](https://mobbin.com/screens/b324dde2-8cf1-4230-9be3-4cb75ed28d6a) (botões) | Botões e degradês; associado a criatividade e IA |
| **Verde** | [Buffer](https://mobbin.com/screens/82fb0dce-e613-44aa-abcd-20c9786e4f1b), [AirOps](https://mobbin.com/screens/5df1c6e4-b94a-4a91-a085-11fe97d5443c), [Mintlify](https://mobbin.com/screens/919845de-7cb0-447b-a10d-18ba6a933ee4) | Buffer (concorrente direto em agendamento de posts) usa verde claro no botão "New" |
| **Laranja / coral** | [Base44](https://mobbin.com/screens/350b8449-49f2-479a-9278-5c2c377d915e), [Etsy](https://mobbin.com/screens/ffbe94ce-ef9a-4cdc-b120-99868002a2fb) | Logo e destaques pontuais |

## Leitura
1. **Tendência forte:** produtos de marketing mais recentes usam **botões neutros escuros** e deixam a cor da marca para o logo, o item ativo e os destaques. A tela fica mais sóbria e o conteúdo (posts, imagens) aparece mais.
2. **Azul-índigo está saturado:** é a cor mais comum entre os produtos SaaS (softwares vendidos por assinatura). Usar a mesma cor da Laura Hub deixaria a Laura Growth parecida com Canny, Featurebase e com a própria Hub.
3. **Roxo já tem papel no nosso sistema:** no [[Design System - Plataforma Laura Growth]], o roxo (`--laura-ai`) marca o que a IA fez. Usar roxo como cor da marca confundiria esse sinal.
4. **Verde também já tem papel:** é a cor de "sucesso". Como cor da marca, um botão verde poderia ser lido como "concluído".
5. **Laranja/coral é pouco usado** no segmento e não conflita com azul (Hub), roxo (IA) nem verde (sucesso). Ponto de cuidado: ficar distante do âmbar usado em "alerta".

## Recomendação
**Cor própria para a Laura Growth, numa família quente (coral)**, com botões principais em grafite, seguindo a tendência neutra.

| Opção | Prós | Contras |
|---|---|---|
| **Recomendada: coral + botões grafite** | Diferencia da Hub e do mercado; não conflita com os sinais de IA, sucesso e alerta; combina com marketing (energia, crescimento) | Exige testar o contraste do texto sobre o coral |
| Mesma cor da Hub | Zero trabalho; reforça a família Laura | Não diferencia as marcas; cor mais comum do segmento |
| Verde ("growth") | Associação direta com crescimento | Conflita com "sucesso"; Buffer já usa |

Como fica nos dois produtos: a família Laura mantém a mesma estrutura, os mesmos tokens e componentes; muda só a cor da marca (Hub = azul, Growth = coral). Ver [[Laura Hub x Laura Growth]].

## Decisão (15/11/2026)
**Aprovado: coral + botões grafite.**

| Token | Código | Contraste sobre branco | Uso |
|---|---|---|---|
| `--laura-primary` | `#1f2430` | 15,5 | Botões principais |
| `--laura-brand` | `#f26b3a` | 3,0 | Logo e detalhes gráficos (nunca texto) |
| `--laura-brand-strong` | `#c2410c` | 5,2 | Texto, links, item ativo |
| `--laura-brand-soft` | `#fff3ed` | — | Fundos de destaque |

- [x] Família de cor decidida
- [x] Códigos definidos e contraste testado (WCAG AA)
- [x] Tokens atualizados no [[Design System - Plataforma Laura Growth]]
- [x] Registrado em [[PRD - Decisões de Negócio]]
- [ ] Ver o coral aplicado em uma tela real (etapa 0) e ajustar o tom se preciso
- [ ] Logo da Laura Growth em coral: a definir

## Relacionado
[[Design System - Plataforma Laura Growth]] · [[Laura Growth - Posicionamento]] · [[Índice - Projetos]]
