package embeddings

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type ApiEmbedder struct {
	url    string
	client *http.Client
}

func NewApi(url string) *ApiEmbedder {
	return &ApiEmbedder{url: url, client: &http.Client{}}
}

func (e *ApiEmbedder) ModelName() string { return "api" }

func (e *ApiEmbedder) EmbedTexts(texts []string) ([][]float32, error) {
	embeddings, err := e.embedRequest(texts)
	if err != nil {
		return nil, err
	}
	return embeddings, nil
}

func (e *ApiEmbedder) EmbedQuery(text string) ([]float32, error) {
	embeddings, err := e.embedRequest([]string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

type embedRequest struct {
	Sentences []string `json:"sentences"`
}

func (e *ApiEmbedder) embedRequest(texts []string) ([][]float32, error) {
	request := &embedRequest{
		Sentences: texts,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := e.client.Post(e.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	var embeddings [][]float32
	if err := json.NewDecoder(response.Body).Decode(&embeddings); err != nil {
		return nil, err
	}
	return embeddings, nil
}
