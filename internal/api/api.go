package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/o0n1x/Sublate/internal/auth"
	"github.com/o0n1x/Sublate/internal/cache"
	"github.com/o0n1x/Sublate/internal/database"
	serr "github.com/o0n1x/sublate-go/errors"
	"github.com/o0n1x/sublate-go/format"
	"github.com/o0n1x/sublate-go/lang"
	"github.com/o0n1x/sublate-go/provider"
	"github.com/redis/go-redis/v9"
)

// constants
const MAXFILESIZE = 50 << 20
const MAXQUERYSIZE = 100

const USERCONTEXTKEY = "user"

type ApiConfig struct {
	DB       *database.Queries
	Redis    *redis.Client
	Platform string
	Clients  map[provider.Provider]provider.Client
	//DeeplClient      *deepl.DeepLClient
	//DeeplClientAPI   string
	AdminCredentials struct {
		Email    string
		Password string
	}
	SECRET_JWT string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
}

// handles all API functions

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *ApiConfig) Login(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		errorRespond(w, 400, "Invalid JSON in the request body")
		return
	}

	user, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		log.Printf("user not found: %v", err)
		errorRespond(w, 401, "Incorrect email or password")
		return
	}

	ok, err := auth.CheckPasswordHash(params.Password, user.HashedPassword.String)
	if !ok {
		log.Printf("password does not match: %v", err)
		errorRespond(w, 401, "Incorrect email or password")
		return
	}

	jwt_token, err := auth.MakeJWT(user.ID, cfg.SECRET_JWT, time.Hour) //TODO: remove hardcoded time limit
	if err != nil {
		log.Printf("Error creating token: %v", err)
		errorRespond(w, 500, "Failed to create token")
		return
	}

	jsonRespond(w, 200, struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		Token string    `json:"token"`
	}{
		ID:    user.ID,
		Email: user.Email,
		Token: jwt_token,
	})

}

func (cfg *ApiConfig) Register(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		errorRespond(w, 400, "Invalid JSON in the request body")
		return
	}

	hashedpass, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return
	}

	// is_admin is set false by default for security, and then updated manually via SQL
	_, err = cfg.DB.CreateUser(context.Background(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: sql.NullString{String: hashedpass, Valid: true},
		IsAdmin:        false,
	})
	if err != nil {
		log.Printf("User Registeration Failed: %v", err)
		errorRespond(w, 500, "User Registeration Failed")
		return
	}

}

func (cfg *ApiConfig) RegisterAdmin() {
	if cfg.AdminCredentials.Email == "None" {
		log.Printf("Initial Admin Registered Cancelled")
		return
	}
	_, err := cfg.DB.GetUserByEmail(context.Background(), cfg.AdminCredentials.Email)
	if err == nil {
		log.Printf("Initial Admin Credentials Already Registered")
		return
	}

	hashedpass, err := auth.HashPassword(cfg.AdminCredentials.Password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return
	}

	_, err = cfg.DB.CreateUser(context.Background(), database.CreateUserParams{
		Email:          cfg.AdminCredentials.Email,
		HashedPassword: sql.NullString{String: hashedpass, Valid: true},
		IsAdmin:        true,
	})
	if err != nil {
		log.Printf("Initial Admin Registeration Failed: %v", err)
		os.Exit(1)
		return
	}
	log.Printf("Initial Admin Credentials Registered Successfully")

}

func (cfg *ApiConfig) GetUsers(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID != "" {
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			log.Printf("Error invalid user ID: %v", err)
			errorRespond(w, 400, "invalid ID")
			return
		}

		user, err := cfg.DB.GetUser(r.Context(), userUUID)
		if err != nil {
			log.Printf("Error retrieving user: %v", err)
			errorRespond(w, 404, "user not found")
			return
		}

		jsonRespond(w, 200, User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
			IsAdmin:   user.IsAdmin,
		})
		return
	}

	limit := 10
	offset := 0
	limit_query := r.URL.Query().Get("limit")
	offset_query := r.URL.Query().Get("offset")

	if limit_query != "" {
		parsed, err := strconv.Atoi(limit_query)
		if err == nil && parsed > 0 && parsed <= MAXQUERYSIZE {
			limit = parsed
		}
	}
	if offset_query != "" {
		parsed, err := strconv.Atoi(offset_query)
		if err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	users, err := cfg.DB.GetUsers(r.Context(), database.GetUsersParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		log.Printf("Error retrieving users: %v", err)
		errorRespond(w, 500, "Failed to retrieve users")
		return
	}

	returnedUsers := []User{}

	for _, user := range users {
		returnedUsers = append(returnedUsers, User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
			IsAdmin:   user.IsAdmin,
		})
	}

	jsonRespond(w, 200, returnedUsers)

}

func (cfg *ApiConfig) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		log.Printf("Error invalid user ID: %v", err)
		errorRespond(w, 400, "invalid ID")
		return
	}

	user, err := cfg.DB.GetUser(r.Context(), userUUID)
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
		errorRespond(w, 404, "user not found")
		return
	}

	type parameters struct {
		Email    *string `json:"email,omitempty"`
		IsAdmin  *bool   `json:"is_admin,omitempty"`
		Password *string `json:"password,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		errorRespond(w, 400, "Invalid JSON in the request body")
		return
	}

	if params.Email == nil {
		params.Email = &user.Email
	}
	if params.IsAdmin == nil {
		params.IsAdmin = &user.IsAdmin
	}

	if params.Password == nil {
		params.Password = &user.HashedPassword.String
	} else {
		hashedpass, err := auth.HashPassword(*params.Password)
		if err != nil {
			log.Printf("Error updating user: %v", err)
			errorRespond(w, 500, "error updating user")
			return
		}
		params.Password = &hashedpass
	}

	newuser, err := cfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userUUID,
		Email:          *params.Email,
		IsAdmin:        *params.IsAdmin,
		HashedPassword: sql.NullString{String: *params.Password, Valid: true},
	})
	if err != nil {
		log.Printf("Error updating user: %v", err)
		errorRespond(w, 500, "error updating user")
		return
	}

	jsonRespond(w, 200, User{
		ID:        newuser.ID,
		UpdatedAt: newuser.UpdatedAt,
		CreatedAt: newuser.CreatedAt,
		Email:     newuser.Email,
		IsAdmin:   newuser.IsAdmin,
	})

}

func (cfg *ApiConfig) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		log.Printf("Error invalid user ID: %v", err)
		errorRespond(w, 400, "invalid ID")
		return
	}

	_, err = cfg.DB.GetUser(r.Context(), userUUID)
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
		errorRespond(w, 404, "user not found")
		return
	}

	err = cfg.DB.DeleteUser(r.Context(), userUUID)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
		errorRespond(w, 500, "error deleting user")
		return
	}

	w.WriteHeader(204)
}

func (cfg *ApiConfig) Translate(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		cfg.translateFile(w, r)
	} else if contentType == "application/json" {
		cfg.translateText(w, r)
	} else {
		http.Error(w, "unsupported content type", http.StatusBadRequest)
	}
}

type Status struct {
	Done    bool   `json:"done"`
	Failed  bool   `json:"failed"`
	Message string `json:"message"`
	Seconds int    `json:"seconds"`
}

func (cfg *ApiConfig) Status(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(USERCONTEXTKEY).(database.User).ID
	docID := r.PathValue("id")
	docUUID, err := uuid.Parse(docID)
	if err != nil {
		log.Printf("Error invalid user ID: %v", err)
		http.Error(w, "incorrect id", http.StatusBadRequest)
		return
	}

	doc_userid, err := cfg.DB.GetDocumentUser(r.Context(), docUUID)
	if err != nil {
		log.Printf("Error getting userid from documentid: %v", err)
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}
	if doc_userid != user {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	//get provider from request
	row, err := cfg.DB.GetDocumentProvider(r.Context(), docUUID)
	providerName := row.Provider
	client, ok := cfg.Clients[provider.Provider(providerName)]
	if !ok {
		http.Error(w, "Provider no longer available", http.StatusInternalServerError) //we cant blame user for provider not being avaialbe at this point, returning 500 is better
		return
	}
	ac, ok := client.(provider.AsyncClient)
	if !ok {
		log.Printf("error getting %v Async provider when checking status: %v", providerName, err)
		http.Error(w, "Provider no longer available", http.StatusInternalServerError)
		return
	}

	//getting document
	document, err := cfg.DB.GetDocument(r.Context(), docUUID)
	if err != nil {
		log.Printf("Error getting document %v: %v", docUUID, err)
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}

	status, err := ac.CheckStatus(r.Context(), provider.AsyncResponse{DocumentID: document.ExternalID, DocumentKey: document.ExternalKey.String})
	if err != nil {
		log.Printf("Error checking status for document %v: %v", docUUID, err)
		http.Error(w, "error checking status", http.StatusInternalServerError)
		return
	}

	if status.Done {
		_, err := cfg.DB.UpdateDocument(r.Context(), database.UpdateDocumentParams{ID: docUUID, Status: "done"})
		if err != nil {
			log.Printf("Error updating document %v: %v", docUUID, err)
			http.Error(w, "error updating status", http.StatusBadRequest)
			return
		}
		jsonRespond(w, 200, Status{
			Done:   true,
			Failed: false,
		})
	} else if status.Failed {
		_, err := cfg.DB.UpdateDocument(r.Context(), database.UpdateDocumentParams{ID: docUUID, Status: "failed"})
		if err != nil {
			log.Printf("Error updating document %v: %v", docUUID, err)
			http.Error(w, "error updating status", http.StatusBadRequest)
			return
		}
		jsonRespond(w, 200, Status{
			Done:    false,
			Failed:  true,
			Message: status.Message,
		})
	} else {
		jsonRespond(w, 200, Status{
			Done:    false,
			Failed:  false,
			Seconds: status.SecondsRemaining,
		})

	}

}

func (cfg *ApiConfig) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	//validate user
	//delete document
	// respond
}

func (cfg *ApiConfig) Result(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(USERCONTEXTKEY).(database.User).ID
	docID := r.PathValue("id")
	docUUID, err := uuid.Parse(docID)
	if err != nil {
		log.Printf("Error invalid user ID: %v", err)
		http.Error(w, "incorrect id", http.StatusBadRequest)
		return
	}

	doc_userid, err := cfg.DB.GetDocumentUser(r.Context(), docUUID)
	if err != nil {
		log.Printf("Error getting userid from documentid: %v", err)
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}
	if doc_userid != user {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	//get provider from request
	row, err := cfg.DB.GetDocumentProvider(r.Context(), docUUID)
	providerName := row.Provider
	client, ok := cfg.Clients[provider.Provider(providerName)]
	if !ok {
		http.Error(w, "Provider no longer available", http.StatusInternalServerError) //we cant blame user for provider not being avaialbe at this point, returning 500 is better
		return
	}
	ac, ok := client.(provider.AsyncClient)
	if !ok {
		log.Printf("error getting %v Async provider when checking status: %v", providerName, err)
		http.Error(w, "Provider no longer available", http.StatusInternalServerError)
		return
	}

	document, err := cfg.DB.GetDocument(r.Context(), docUUID)
	if err != nil {
		log.Printf("Error getting document %v: %v", docUUID, err)
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}

	if document.Status != "done" && document.Status != "error" {
		http.Error(w, "document not ready yet", http.StatusBadRequest)
		return
	}

	result, err := ac.GetResult(r.Context(), provider.AsyncResponse{DocumentID: document.ExternalID, DocumentKey: document.ExternalKey.String})
	if err != nil {
		log.Printf("Error getting result for document %v: %v", docUUID, err)
		http.Error(w, "error getting result for document", http.StatusInternalServerError)
		return
	}

	fileRespond(w, result.Binary, document.Filename.String)

}

// log api to query logs table
func (cfg *ApiConfig) Logs(w http.ResponseWriter, r *http.Request) {
	//jst like users api endpoint
}

// metrics api endpoint for promethius in the future
func (cfg *ApiConfig) Metrics(w http.ResponseWriter, r *http.Request) {
	//needs a singleton that records metrics and a function that does the counting

}

//
//Helper Functions
//

// TODO: limit how large the cache can be. atm even a 1GB file will be cached
func (cfg *ApiConfig) translateFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MAXFILESIZE)

	providerName := r.FormValue("provider")
	client, ok := cfg.Clients[provider.Provider(providerName)]
	if !ok {
		http.Error(w, "invalid provider", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if r.FormValue("target_lang") == "" {
		http.Error(w, "invalid form no target language", http.StatusBadRequest)
		return
	}

	reqDB, err := cfg.DB.CreateRequest(r.Context(), database.CreateRequestParams{
		Provider: providerName,
		ReqType:  format.File.String(),
		FromLang: r.FormValue("source_lang"),
		ToLang:   r.FormValue("target_lang"),
		UserID:   r.Context().Value(USERCONTEXTKEY).(database.User).ID,
	})
	if err != nil {
		log.Printf("Error creating request: %v", err)
		errorRespond(w, 500, "request creation failed")
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

	//TODO: caching not connected to DB yet
	cached, hit, err := cache.GetCache(r.Context(), cfg.Redis, provider.Provider(providerName), req)
	if err != nil {
		log.Printf("cache error: %v", err)
	}
	if hit {
		_, err := cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: true, Cached: true, Error: sql.NullString{String: "", Valid: false}, RequestID: reqDB.ID})
		if err != nil { //no need to return as logging errors should not affect users
			log.Printf("Error creating Log: %v", err)
		}

		log.Print("Cache HIT")
		w.Header().Set("X-Cache", "HIT")
		fileRespond(w, cached.Binary, req.FileName)
		return
	}

	ac, ok := client.(provider.AsyncClient)
	if ok {
		res, err := ac.AsyncTranslate(r.Context(), req)
		if err != nil {
			_, err := cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: false, Cached: false, Error: sql.NullString{String: err.Error(), Valid: true}, RequestID: reqDB.ID})
			if err != nil { //no need to return as logging errors should not affect users
				log.Printf("Error creating Log: %v", err)
			}

			log.Printf("Error translating: %v", err)
			var tErr *serr.TranslateError
			if errors.As(err, &tErr) {

				switch tErr.Code {
				case serr.ErrInvalidLanguage:
					http.Error(w, "Invalid Language", http.StatusBadRequest)
					return
				case serr.ErrInvalidRequest:
					break // if invalidRequest is returned then fallback to syncclient translation
				case serr.ErrProviderAPI:
					http.Error(w, "Provider Error", http.StatusBadGateway)
					return
				default:
					http.Error(w, "Error translating", http.StatusInternalServerError)
					return
				}
			}
		} else {
			_, err := cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: true, Cached: false, Error: sql.NullString{String: "", Valid: false}, RequestID: reqDB.ID})
			if err != nil { //no need to return as logging errors should not affect users
				log.Printf("Error creating Log: %v", err)
			}
			//TODO respond instead with a jobID
			var exKey sql.NullString
			if res.DocumentKey == "" {
				exKey = sql.NullString{String: res.DocumentKey, Valid: false}
			} else {
				exKey = sql.NullString{String: res.DocumentKey, Valid: true}
			}

			doc, err := cfg.DB.CreateDocument(r.Context(), database.CreateDocumentParams{
				ExternalID:  res.DocumentID,
				ExternalKey: exKey,
				Filename:    sql.NullString{String: req.FileName, Valid: true},
				Status:      "pending",
				RequestID:   reqDB.ID,
			})
			if err != nil {
				log.Printf("Error creating document: %v", err)
				http.Error(w, "Error creating document", http.StatusInternalServerError)
				return
			}
			result := struct {
				DocumentID uuid.UUID `json:"document_id"`
			}{
				DocumentID: doc.ID,
			}
			jsonRespond(w, 202, result)
			return
		}
	}

	sc, ok := client.(provider.SyncClient)
	if ok {
		res, err := sc.Translate(r.Context(), req)
		if err != nil {
			_, err := cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: false, Cached: false, Error: sql.NullString{String: err.Error(), Valid: true}, RequestID: reqDB.ID})
			if err != nil { //no need to return as logging errors should not affect users
				log.Printf("Error creating Log: %v", err)
			}
			log.Printf("Error translating: %v", err)
			var tErr *serr.TranslateError
			if errors.As(err, &tErr) {
				switch tErr.Code {
				case serr.ErrInvalidLanguage:
					http.Error(w, "Invalid Language", http.StatusBadRequest)
				case serr.ErrInvalidRequest:
					http.Error(w, "Invalid Request", http.StatusBadRequest)
				case serr.ErrProviderAPI:
					http.Error(w, "Provider Error", http.StatusBadGateway)
				default:
					http.Error(w, "Error translating", http.StatusInternalServerError)
				}
				return
			}
			http.Error(w, "Error translating", http.StatusInternalServerError) // fallback incase somehow err is not serr error
			log.Printf("Error translating: %v", err)
			return
		}

		_, err = cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: true, Cached: false, Error: sql.NullString{String: "", Valid: false}, RequestID: reqDB.ID})
		if err != nil { //no need to return as logging errors should not affect users
			log.Printf("Error creating Log: %v", err)
		}

		err = cache.SetCache(r.Context(), cfg.Redis, provider.Provider(providerName), req, res)
		if err != nil {
			log.Printf("cache set error: %v", err)
		}

		w.Header().Set("X-Cache", "MISS")
		fileRespond(w, res.Binary, req.FileName)
	}

}

func (cfg *ApiConfig) translateText(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Provider   string   `json:"provider"`
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

	providerName := params.Provider
	client, ok := cfg.Clients[provider.Provider(providerName)]
	if !ok {
		http.Error(w, "invalid provider", http.StatusBadRequest)
		return
	}

	reqDB, err := cfg.DB.CreateRequest(r.Context(), database.CreateRequestParams{
		Provider: providerName,
		ReqType:  format.Text.String(),
		FromLang: params.SourceLang,
		ToLang:   params.TargetLang,
		UserID:   r.Context().Value(USERCONTEXTKEY).(database.User).ID,
	})
	if err != nil {
		log.Printf("Error creating request: %v", err)
		errorRespond(w, 500, "request creation failed")
		return
	}

	req := provider.Request{
		ReqType: format.Text,
		Text:    params.Text,
		From:    lang.Language(params.SourceLang),
		To:      lang.Language(params.TargetLang),
	}

	cached, hit, err := cache.GetCache(r.Context(), cfg.Redis, provider.Provider(providerName), req)
	if err != nil {
		log.Printf("cache error: %v", err)
	}
	if hit {
		_, err := cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: true, Cached: true, Error: sql.NullString{String: "", Valid: false}, RequestID: reqDB.ID})
		if err != nil { //no need to return as logging errors should not affect users
			log.Printf("Error creating Log: %v", err)
		}
		log.Print("Cache HIT")
		w.Header().Set("X-Cache", "HIT")
		textRespond(w, cached.Text)
		return
	}

	//Assumption that all provider is Sync with text translation
	sc, ok := client.(provider.SyncClient)
	if !ok {
		http.Error(w, "invalid provider", http.StatusBadRequest)
		return
	}

	res, err := sc.Translate(r.Context(), req)
	if err != nil {
		_, err := cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: false, Cached: false, Error: sql.NullString{String: err.Error(), Valid: true}, RequestID: reqDB.ID})
		if err != nil { //no need to return as logging errors should not affect users
			log.Printf("Error creating Log: %v", err)
		}
		var tErr *serr.TranslateError
		if errors.As(err, &tErr) {
			switch tErr.Code {
			case serr.ErrInvalidLanguage:
				http.Error(w, "Invalid Language", http.StatusBadRequest)
			case serr.ErrInvalidRequest:
				http.Error(w, "Invalid Request", http.StatusBadRequest)
			case serr.ErrProviderAPI:
				http.Error(w, "Provider Error", http.StatusBadGateway)
			default:
				http.Error(w, "Error translating", http.StatusInternalServerError)
			}
			log.Printf("Error translating: %v", err)
			return
		}
		http.Error(w, "Error translating", http.StatusInternalServerError) // fallback incase somehow err is not serr error
		log.Printf("Error translating: %v", err)
		return
	}

	err = cache.SetCache(r.Context(), cfg.Redis, provider.Provider(providerName), req, res)
	if err != nil {
		log.Printf("cache set error: %v", err)
	}

	_, err = cfg.DB.CreateLog(r.Context(), database.CreateLogParams{IsSuccessful: true, Cached: false, Error: sql.NullString{String: "", Valid: false}, RequestID: reqDB.ID})
	if err != nil { //no need to return as logging errors should not affect users
		log.Printf("Error creating Log: %v", err)
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

func fileRespond(w http.ResponseWriter, binary []byte, filename string) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"translated_%s\"", filename))
	w.Write(binary)
}

func errorRespond(w http.ResponseWriter, code int, msg string) {
	type returnErr struct {
		Error string `json:"error"`
	}

	respBody := returnErr{
		Error: msg,
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func jsonRespond(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func isFileAllowedDeepl(filename string) bool {
	allowed := map[string]bool{
		".srt":  true,
		".txt":  true,
		".docx": true}

	ext := filepath.Ext(filename)

	return allowed[ext]
}

//
// Middleware
//

func (cfg *ApiConfig) MiddlewareIsUser(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("Error parsing header: %v", err)
			errorRespond(w, 401, "Token missing or invalid, Please Login First")
			return
		}
		userid, err := auth.ValidateJWT(token, cfg.SECRET_JWT)
		if err != nil {
			log.Printf("Error validating token: %v", err)
			errorRespond(w, 401, "Token missing or invalid, Please Login First")
			return
		}
		user, err := cfg.DB.GetUser(r.Context(), userid)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			errorRespond(w, 401, "Token missing or invalid, Please Login First")
			return
		}
		ctx := context.WithValue(r.Context(), USERCONTEXTKEY, user)
		next(w, r.WithContext(ctx))
	}
}
func (cfg *ApiConfig) MiddlewareIsAdmin(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("Error parsing header: %v", err)
			errorRespond(w, 401, "Token missing or invalid")
			return
		}
		userid, err := auth.ValidateJWT(token, cfg.SECRET_JWT)
		if err != nil {
			log.Printf("Error validating token: %v", err)
			errorRespond(w, 401, "Token missing or invalid")
			return
		}
		user, err := cfg.DB.GetUser(r.Context(), userid)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			errorRespond(w, 401, "Token missing or invalid")
			return
		}
		if !user.IsAdmin {
			log.Printf("user %v attempted an admin action", user.ID)
			errorRespond(w, 403, "Forbidden")
			return
		}
		next(w, r)
	}
}
