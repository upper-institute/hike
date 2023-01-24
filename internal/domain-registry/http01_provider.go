package domainregistry

import (
	"net/http"
	"strings"

	"github.com/go-acme/lego/v4/challenge/http01"
	"go.uber.org/zap"
)

type http01Challenge struct {
	domain  string
	keyAuth string
}

type HTTP01Provider struct {
	tokenMap map[string]*http01Challenge
	logger   *zap.SugaredLogger
}

func NewHTTP01Provider(logger *zap.SugaredLogger) *HTTP01Provider {
	return &HTTP01Provider{
		tokenMap: make(map[string]*http01Challenge),
		logger:   logger.With("component", "http01-provider"),
	}
}

func (h *HTTP01Provider) AddToServeMux(serveMux *http.ServeMux) {
	serveMux.Handle(http01.ChallengePath(""), h)
}

func (h *HTTP01Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")

	urlPath := strings.TrimRight(r.URL.Path, "/")

	sep := strings.LastIndex(urlPath, "/")

	if sep < 0 {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	token := urlPath[sep+1:]

	challenge, ok := h.tokenMap[token]
	if !ok {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet && strings.HasPrefix(r.Host, challenge.domain) {
		_, err := w.Write([]byte(challenge.keyAuth))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.logger.Warnf("Received request for domain %s with method %s but the domain did not match any challenge. Please ensure you are passing the Host header properly.", r.Host, r.Method)
	_, err := w.Write([]byte("TEST"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func (h *HTTP01Provider) Present(domain, token, keyAuth string) error {
	h.tokenMap[token] = &http01Challenge{domain, keyAuth}
	return nil
}

func (h *HTTP01Provider) CleanUp(domain, token, keyAuth string) error {
	delete(h.tokenMap, token)
	return nil
}
