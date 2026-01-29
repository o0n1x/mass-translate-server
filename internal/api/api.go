package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/o0n1x/mass-translate-package/format"
	"github.com/o0n1x/mass-translate-package/lang"
	"github.com/o0n1x/mass-translate-package/provider"
	"github.com/o0n1x/mass-translate-package/provider/deepl"
	"github.com/o0n1x/mass-translate-package/translator"
	"github.com/o0n1x/mass-translate-server/internal/cache"
	"github.com/o0n1x/mass-translate-server/internal/database"
	"github.com/redis/go-redis/v9"
)

// constants
const MAXFILESIZE = 50 << 20

type ApiConfig struct {
	DB             *database.Queries
	Redis          *redis.Client
	Platform       string
	DeeplClient    *deepl.DeepLClient
	DeeplClientAPI string
}

// handles all API functions

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *ApiConfig) DeeplTranslate(w http.ResponseWriter, r *http.Request) {

	if cfg.DeeplClient == nil {
		generalizedclient, _ := provider.GetClient(provider.DeepL, cfg.DeeplClientAPI)
		cfg.DeeplClient = generalizedclient.(*deepl.DeepLClient)
	}

	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		cfg.fileTranslateHelper(w, r)
	} else if contentType == "application/json" {
		cfg.textTranslateHelper(w, r)
	} else {
		http.Error(w, "unsupported content type", http.StatusBadRequest)
	}
}

func (cfg *ApiConfig) textTranslateHelper(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Text       []string `json:"text"`
		SourceLang string   `json:"source_lang"`
		TargetLang string   `json:"target_lang"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		http.Error(w, "Invalid JSON in the request body", http.StatusBadRequest)
		log.Printf("Error decoding parameters: %s", err)
		return
	}

	req := provider.Request{
		ReqType: format.Text,
		Text:    params.Text,
		From:    lang.Language(params.SourceLang),
		To:      lang.Language(params.TargetLang),
	}

	cached, hit, err := cache.GetCache(r.Context(), cfg.Redis, provider.DeepL, req)
	if err != nil {
		log.Printf("cache error: %v", err)
	}
	if hit {
		log.Print("Cache HIT")
		w.Header().Set("X-Cache", "HIT")
		textRespond(w, cached.Text)
		return
	}

	res, err := cfg.DeeplClient.Translate(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid Source Language") {
			http.Error(w, "Error translating: Invalid Source Language", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "Invalid Target Language") {
			http.Error(w, "Error translating: Invalid Target Language", http.StatusBadRequest)
		} else {
			http.Error(w, "Error translating", http.StatusInternalServerError)
		}

		log.Printf("Error translating: %v", err)
		return
	}

	err = cache.SetCache(r.Context(), cfg.Redis, provider.DeepL, req, res)
	if err != nil {
		log.Printf("cache set error: %v", err)
	}

	w.Header().Set("X-Cache", "MISS")
	textRespond(w, res.Text)

}
func textRespond(w http.ResponseWriter, text []string) {
	type TextResponse struct {
		Translations []string `json:"translation"`
	}
	textres := TextResponse{Translations: text}
	dat, err := json.Marshal(textres)
	if err != nil {
		http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
		log.Printf("Error marshalling JSON: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}

// TODO: limit how large the cache can be. atm even a 1GB file will be cached
func (cfg *ApiConfig) fileTranslateHelper(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MAXFILESIZE)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !isFileAllowedDeepl(header.Filename) {
		http.Error(w, "invalid file type", http.StatusBadRequest)
		return
	}

	if r.FormValue("target_lang") == "" {
		http.Error(w, "invalid form no target language", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		log.Printf("Error reading file: %v", err)
		return
	}

	req := provider.Request{
		ReqType:  format.File,
		Binary:   data,
		FileName: header.Filename,
		From:     lang.Language(r.FormValue("source_lang")),
		To:       lang.Language(r.FormValue("target_lang")),
	}

	cached, hit, err := cache.GetCache(r.Context(), cfg.Redis, provider.DeepL, req)
	if err != nil {
		log.Printf("cache error: %v", err)
	}
	if hit {
		log.Print("Cache HIT")
		w.Header().Set("X-Cache", "HIT")
		fileRespond(w, cached.Binary, req.FileName)
		return
	}

	res, err := translator.Translate(r.Context(), req, cfg.DeeplClient)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid Source Language") {
			http.Error(w, "Error translating: Invalid Source Language", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "Invalid Target Language") {
			http.Error(w, "Error translating: Invalid Target Language", http.StatusBadRequest)
		} else {
			http.Error(w, "Error translating", http.StatusInternalServerError)
		}
		log.Printf("Error translating: %v", err)
		return
	}

	err = cache.SetCache(r.Context(), cfg.Redis, provider.DeepL, req, res)
	if err != nil {
		log.Printf("cache set error: %v", err)
	}

	w.Header().Set("X-Cache", "MISS")
	fileRespond(w, res.Binary, req.FileName)

}

func fileRespond(w http.ResponseWriter, binary []byte, filename string) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"translated_%s\"", filename))
	w.Write(binary)
}

func isFileAllowedDeepl(filename string) bool {
	allowed := map[string]bool{
		".srt":  true,
		".txt":  true,
		".docx": true}

	ext := filepath.Ext(filename)

	return allowed[ext]
}
