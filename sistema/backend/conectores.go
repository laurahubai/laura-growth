package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	versaoAnthropic      = "2023-06-01"
	limiteRespostaPadrao = int64(2 << 20)  // 2 MB para listagens e texto
	limiteRespostaImagem = int64(20 << 20) // 20 MB para geração de imagem em base64
	maxPaginasAnthropic  = 20
	maxTokensFornecedor  = 10_000_000
)

// PNG com um círculo azul sobre fundo branco para respostas previsíveis do conector simulado.
var pngAzulSimulado = func() string {
	const lado = 64
	img := image.NewRGBA(image.Rect(0, 0, lado, lado))
	branco := color.RGBA{255, 255, 255, 255}
	azul := color.RGBA{37, 99, 235, 255}
	centro := float64(lado-1) / 2
	raio2 := 20.0 * 20.0
	for y := 0; y < lado; y++ {
		for x := 0; x < lado; x++ {
			dx, dy := float64(x)-centro, float64(y)-centro
			if dx*dx+dy*dy <= raio2 {
				img.SetRGBA(x, y, azul)
			} else {
				img.SetRGBA(x, y, branco)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}()

type respostaTextoIA struct {
	Texto         string
	TokensEntrada int
	TokensSaida   int
}

type respostaImagemIA struct {
	ImagemDataUrl       string
	TokensEntrada       int
	TokensEntradaImagem int
	TokensSaida         int
	Imagens             int
}

type conectorIA interface {
	listarModelos(ctx context.Context) ([]string, error)
	gerarTexto(ctx context.Context, modelo, prompt string) (respostaTextoIA, error)
	gerarImagem(ctx context.Context, modelo, prompt string) (respostaImagemIA, error)
}

type erroFornecedor struct {
	codigo   string
	mensagem string
}

func (e *erroFornecedor) Error() string { return e.mensagem }

func novoConector(tipo, endereco, chave string, cliente *http.Client) (conectorIA, error) {
	switch tipo {
	case "simulado":
		return &conectorSimulado{}, nil
	case "openai", "compativel_openai":
		return &conectorOpenAI{endereco: strings.TrimRight(endereco, "/"), chave: chave, cliente: cliente}, nil
	case "anthropic":
		return &conectorAnthropic{endereco: strings.TrimRight(endereco, "/"), chave: chave, cliente: cliente}, nil
	default:
		return nil, &erroFornecedor{codigo: "fornecedor_erro", mensagem: "Tipo de fornecedor não suportado."}
	}
}

// ---- Conector simulado (sem rede; previsível para testes e homologação) ----

type conectorSimulado struct{}

func (c *conectorSimulado) listarModelos(_ context.Context) ([]string, error) {
	return []string{"simulado-texto", "simulado-imagem"}, nil
}

func (c *conectorSimulado) gerarTexto(_ context.Context, modelo, prompt string) (respostaTextoIA, error) {
	if strings.Contains(strings.ToLower(modelo), "falha") {
		return respostaTextoIA{}, &erroFornecedor{
			codigo:   "fornecedor_erro",
			mensagem: "O fornecedor simulado falhou para este modelo de teste.",
		}
	}
	return respostaTextoIA{
		Texto:         fmt.Sprintf("Resposta simulada (%s): %s", modelo, prompt),
		TokensEntrada: 12,
		TokensSaida:   18,
	}, nil
}

func (c *conectorSimulado) gerarImagem(_ context.Context, modelo, _ string) (respostaImagemIA, error) {
	if strings.Contains(strings.ToLower(modelo), "falha") {
		return respostaImagemIA{}, &erroFornecedor{
			codigo:   "fornecedor_erro",
			mensagem: "O fornecedor simulado falhou para este modelo de teste.",
		}
	}
	return respostaImagemIA{
		ImagemDataUrl:       pngAzulSimulado,
		TokensEntrada:       12,
		TokensEntradaImagem: 0,
		TokensSaida:         1258,
		Imagens:             1,
	}, nil
}

// ---- Conector OpenAI e compatíveis ----

type conectorOpenAI struct {
	endereco string
	chave    string
	cliente  *http.Client
}

func (c *conectorOpenAI) cabecalhos() map[string]string {
	return map[string]string{"Authorization": "Bearer " + c.chave}
}

func (c *conectorOpenAI) listarModelos(ctx context.Context) ([]string, error) {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := fazerPedidoJSON(ctx, c.cliente, http.MethodGet, c.endereco+"/models", c.cabecalhos(), nil, &resp, limiteRespostaPadrao); err != nil {
		return nil, err
	}
	return extrairIDsModelos(len(resp.Data), func(i int) string { return resp.Data[i].ID }), nil
}

func (c *conectorOpenAI) gerarTexto(ctx context.Context, modelo, prompt string) (respostaTextoIA, error) {
	corpo := map[string]any{
		"model":    modelo,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := fazerPedidoJSON(ctx, c.cliente, http.MethodPost, c.endereco+"/chat/completions", c.cabecalhos(), corpo, &resp, limiteRespostaPadrao); err != nil {
		return respostaTextoIA{}, err
	}
	if len(resp.Choices) == 0 {
		return respostaTextoIA{}, &erroFornecedor{codigo: "fornecedor_erro", mensagem: "O fornecedor não retornou nenhuma resposta."}
	}
	if err := validarTokensFornecedor(resp.Usage.PromptTokens, resp.Usage.CompletionTokens); err != nil {
		return respostaTextoIA{}, err
	}
	return respostaTextoIA{
		Texto:         resp.Choices[0].Message.Content,
		TokensEntrada: resp.Usage.PromptTokens,
		TokensSaida:   resp.Usage.CompletionTokens,
	}, nil
}

func (c *conectorOpenAI) gerarImagem(ctx context.Context, modelo, prompt string) (respostaImagemIA, error) {
	corpo := map[string]any{
		"model":  modelo,
		"prompt": prompt,
		"n":      1,
	}
	// Apenas modelos dall-e-* aceitam response_format; os demais (gpt-image-*, chatgpt-image-*) devolvem b64_json por padrão e recusam o parâmetro.
	if strings.HasPrefix(strings.ToLower(modelo), "dall-e-") {
		corpo["response_format"] = "b64_json"
	}
	var resp struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
		Usage struct {
			InputTokens        int `json:"input_tokens"`
			OutputTokens       int `json:"output_tokens"`
			InputTokensDetails *struct {
				TextTokens  *int `json:"text_tokens"`
				ImageTokens *int `json:"image_tokens"`
			} `json:"input_tokens_details"`
		} `json:"usage"`
	}
	if err := fazerPedidoJSON(ctx, c.cliente, http.MethodPost, c.endereco+"/images/generations", c.cabecalhos(), corpo, &resp, limiteRespostaImagem); err != nil {
		return respostaImagemIA{}, err
	}
	if len(resp.Data) == 0 || resp.Data[0].B64JSON == "" {
		return respostaImagemIA{}, &erroFornecedor{codigo: "fornecedor_erro", mensagem: "O fornecedor não retornou nenhuma imagem."}
	}
	brutoImg, err := base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
	if err != nil || len(brutoImg) == 0 {
		return respostaImagemIA{}, &erroFornecedor{codigo: "fornecedor_erro", mensagem: "O fornecedor retornou uma imagem inválida."}
	}
	mime := http.DetectContentType(brutoImg)
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
		return respostaImagemIA{}, &erroFornecedor{codigo: "fornecedor_erro", mensagem: "O fornecedor retornou um formato de imagem não aceito."}
	}
	if err := validarTokensFornecedor(resp.Usage.InputTokens, resp.Usage.OutputTokens); err != nil {
		return respostaImagemIA{}, err
	}
	tokensTexto := resp.Usage.InputTokens
	tokensImagem := 0
	if d := resp.Usage.InputTokensDetails; d != nil && (d.TextTokens != nil || d.ImageTokens != nil) {
		tokensTexto = 0
		if d.TextTokens != nil {
			tokensTexto = *d.TextTokens
		}
		if d.ImageTokens != nil {
			tokensImagem = *d.ImageTokens
		}
		if err := validarTokensFornecedor(tokensTexto, tokensImagem); err != nil {
			return respostaImagemIA{}, err
		}
	}
	return respostaImagemIA{
		ImagemDataUrl:       "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(brutoImg),
		TokensEntrada:       tokensTexto,
		TokensEntradaImagem: tokensImagem,
		TokensSaida:         resp.Usage.OutputTokens,
		Imagens:             len(resp.Data),
	}, nil
}

// ---- Conector Anthropic ----

type conectorAnthropic struct {
	endereco string
	chave    string
	cliente  *http.Client
}

func (c *conectorAnthropic) cabecalhos() map[string]string {
	return map[string]string{
		"x-api-key":         c.chave,
		"anthropic-version": versaoAnthropic,
	}
}

func (c *conectorAnthropic) listarModelos(ctx context.Context) ([]string, error) {
	vistos := map[string]bool{}
	var todos []string
	afterID := ""

	for pagina := 0; pagina < maxPaginasAnthropic; pagina++ {
		urlDestino := c.endereco + "/models?limit=1000"
		if afterID != "" {
			urlDestino += "&after_id=" + url.QueryEscape(afterID)
		}
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := fazerPedidoJSON(ctx, c.cliente, http.MethodGet, urlDestino, c.cabecalhos(), nil, &resp, limiteRespostaPadrao); err != nil {
			return nil, err
		}
		for _, item := range resp.Data {
			id := strings.TrimSpace(item.ID)
			if id != "" && !vistos[id] {
				vistos[id] = true
				todos = append(todos, id)
			}
		}
		if !resp.HasMore {
			break
		}
		proximo := strings.TrimSpace(resp.LastID)
		if proximo == "" && len(resp.Data) > 0 {
			proximo = strings.TrimSpace(resp.Data[len(resp.Data)-1].ID)
		}
		if proximo == "" || proximo == afterID {
			break
		}
		afterID = proximo
	}
	return todos, nil
}

func (c *conectorAnthropic) gerarTexto(ctx context.Context, modelo, prompt string) (respostaTextoIA, error) {
	corpo := map[string]any{
		"model":      modelo,
		"max_tokens": 1024,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	}
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := fazerPedidoJSON(ctx, c.cliente, http.MethodPost, c.endereco+"/messages", c.cabecalhos(), corpo, &resp, limiteRespostaPadrao); err != nil {
		return respostaTextoIA{}, err
	}
	var partes []string
	for _, b := range resp.Content {
		if b.Type == "text" && b.Text != "" {
			partes = append(partes, b.Text)
		}
	}
	if len(partes) == 0 {
		return respostaTextoIA{}, &erroFornecedor{codigo: "fornecedor_erro", mensagem: "O fornecedor não retornou texto."}
	}
	if err := validarTokensFornecedor(resp.Usage.InputTokens, resp.Usage.OutputTokens); err != nil {
		return respostaTextoIA{}, err
	}
	return respostaTextoIA{
		Texto:         strings.Join(partes, ""),
		TokensEntrada: resp.Usage.InputTokens,
		TokensSaida:   resp.Usage.OutputTokens,
	}, nil
}

func (c *conectorAnthropic) gerarImagem(_ context.Context, _, _ string) (respostaImagemIA, error) {
	return respostaImagemIA{}, &erroFornecedor{
		codigo:   "fornecedor_erro",
		mensagem: "A Anthropic não gera imagens.",
	}
}

// ---- Auxiliares comuns ----

func validarTokensFornecedor(entrada, saida int) error {
	if entrada < 0 || entrada > maxTokensFornecedor || saida < 0 || saida > maxTokensFornecedor {
		return &erroFornecedor{
			codigo:   "fornecedor_erro",
			mensagem: "O fornecedor informou uma quantidade de tokens fora do limite permitido.",
		}
	}
	return nil
}

func extrairIDsModelos(total int, pegar func(int) string) []string {
	vistos := map[string]bool{}
	out := make([]string, 0, total)
	for i := 0; i < total; i++ {
		id := strings.TrimSpace(pegar(i))
		if id == "" || vistos[id] {
			continue
		}
		vistos[id] = true
		out = append(out, id)
	}
	return out
}

// fazerPedidoJSON executa uma chamada JSON para fornecedores externos e traduz falhas
// para códigos estáveis, sem nunca expor chaves nem o corpo bruto de erro do fornecedor.
func fazerPedidoJSON(ctx context.Context, cliente *http.Client, metodo, urlDestino string, cabecalhos map[string]string, corpo any, destino any, limiteBytes int64) error {
	if cliente == nil {
		return errors.New("cliente HTTP não configurado")
	}
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, urlDestino, leitor)
	if err != nil {
		return &erroFornecedor{codigo: "fornecedor_erro", mensagem: "Não foi possível preparar o pedido para o fornecedor."}
	}
	for k, v := range cabecalhos {
		req.Header.Set(k, v)
	}
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	res, err := cliente.Do(req)
	if err != nil {
		return &erroFornecedor{
			codigo:   "fornecedor_indisponivel",
			mensagem: "Não foi possível conectar ao fornecedor de IA. Tente de novo em instantes.",
		}
	}
	defer res.Body.Close()

	dados, err := io.ReadAll(io.LimitReader(res.Body, limiteBytes+1))
	if err != nil {
		return &erroFornecedor{
			codigo:   "fornecedor_indisponivel",
			mensagem: "A resposta do fornecedor foi interrompida antes de terminar.",
		}
	}
	if int64(len(dados)) > limiteBytes {
		return &erroFornecedor{
			codigo:   "fornecedor_erro",
			mensagem: "resposta do fornecedor maior que o permitido",
		}
	}

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return &erroFornecedor{
			codigo:   "fornecedor_recusou_chave",
			mensagem: "O fornecedor recusou a chave de API informada. Confira a chave cadastrada.",
		}
	}
	if res.StatusCode >= 500 {
		return &erroFornecedor{
			codigo:   "fornecedor_indisponivel",
			mensagem: fmt.Sprintf("O fornecedor de IA está indisponível no momento (código %d).", res.StatusCode),
		}
	}
	if res.StatusCode == http.StatusTooManyRequests {
		return &erroFornecedor{
			codigo:   "fornecedor_indisponivel",
			mensagem: "O fornecedor pediu para aguardar por limite de requisições (429).",
		}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &erroFornecedor{
			codigo:   "fornecedor_erro",
			mensagem: fmt.Sprintf("O fornecedor retornou um erro (%d).", res.StatusCode),
		}
	}
	if err := json.Unmarshal(dados, destino); err != nil {
		return &erroFornecedor{
			codigo:   "fornecedor_erro",
			mensagem: "O fornecedor respondeu em um formato inesperado.",
		}
	}
	return nil
}
