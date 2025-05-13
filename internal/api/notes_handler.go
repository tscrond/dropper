package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
)

type NoteContent struct {
	Content string `json:"content"`
}

func (s *APIServer) fileNotesHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPut:
		s.editFileNotes(w, r)
	case http.MethodGet:
		s.getFileNotes(w, r)
	}
}

func (s *APIServer) editFileNotes(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	// parse data of logged in user
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"response": "authorization_failed",
			"code":     http.StatusForbidden,
		})
		return
	}

	md5Checksum := chi.URLParam(r, "checksum")

	var req NoteContent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "no content",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	sanitizedNoteContent := s.backendConfig.HTMLSanitizationPolicy.Sanitize(req.Content)

	id, err := s.repository.Queries.GetFileFromChecksum(ctx, md5Checksum)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "cannot get file",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	log.Println("Sanitized note content: ", sanitizedNoteContent)

	if utf8.RuneCountInString(sanitizedNoteContent) > 500 {
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"response": "too_many_characters",
			"code":     http.StatusForbidden,
		})
		return
	}

	if _, err = s.repository.Queries.UpdateNoteForFile(
		ctx,
		sqlc.UpdateNoteForFileParams{
			UserID:  sql.NullString{Valid: true, String: authUserData.Id},
			FileID:  sql.NullInt32{Valid: true, Int32: id},
			Content: sanitizedNoteContent,
		},
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "cannot_update",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"response": "created_note",
		"code":     http.StatusOK,
		"note":     sanitizedNoteContent,
	})
}

func (s *APIServer) getFileNotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	// parse data of logged in user
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"response": "authorization_failed",
			"code":     http.StatusForbidden,
		})
		return
	}

	md5Checksum := chi.URLParam(r, "checksum")
	if md5Checksum == "" {
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"response": "checksum_empty",
			"code":     http.StatusForbidden,
		})
		return
	}

	fileId, err := s.repository.Queries.GetFileFromChecksum(ctx, md5Checksum)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "error querying file id",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	note, err := s.repository.Queries.GetNoteForFileById(ctx, sqlc.GetNoteForFileByIdParams{
		UserID: sql.NullString{Valid: true, String: authUserData.Id},
		FileID: sql.NullInt32{Valid: true, Int32: fileId},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "error getting note",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"code":    http.StatusOK,
		"content": note.Content,
	})
}
