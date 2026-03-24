package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/tscrond/dropper/internal/pathutil"
	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
	pkg "github.com/tscrond/dropper/pkg"
)

// getFilesTree handles GET /files/tree?path=<folderPath>
// Returns the direct-child files and subfolders under the given path.
// If path is empty, lists everything at the root level (all folders).
func (s *APIServer) getFilesTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
		return
	}

	ctx := r.Context()
	authUserData, ok := ctx.Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	folderPath := r.URL.Query().Get("path")

	ownerID := sql.NullString{Valid: true, String: authUserData.Id}

	var allFiles []sqlc.File
	var err error

	if folderPath == "" {
		// Root level: get all files to derive top-level folders
		allFiles, err = s.repository.Queries.GetFilesByOwner(ctx, ownerID)
	} else {
		// Specific folder: get all files under this prefix
		allFiles, err = s.repository.Queries.GetFilesByOwnerAndPathPrefix(ctx, sqlc.GetFilesByOwnerAndPathPrefixParams{
			OwnerGoogleID: ownerID,
			FolderPath:    folderPath,
		})
	}
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	// Separate direct files from subfolders
	type FileEntry struct {
		Name     string `json:"name"`
		FileType string `json:"file_type"`
		Size     int64  `json:"size"`
		Checksum string `json:"md5_checksum"`
	}

	prefix := ""
	if folderPath != "" {
		prefix = folderPath + "/"
	}

	var directFiles []FileEntry
	subFolderSet := make(map[string]struct{})

	for _, f := range allFiles {
		if !strings.HasPrefix(f.FileName, prefix) {
			continue
		}
		rest := strings.TrimPrefix(f.FileName, prefix)
		idx := strings.Index(rest, "/")
		if idx < 0 {
			// Direct file
			directFiles = append(directFiles, FileEntry{
				Name:     f.FileName,
				FileType: f.FileType.String,
				Size:     f.Size.Int64,
				Checksum: f.Md5Checksum,
			})
		} else {
			// Sub-folder
			subFolderSet[rest[:idx]] = struct{}{}
		}
	}

	subFolders := make([]string, 0, len(subFolderSet))
	for name := range subFolderSet {
		subFolders = append(subFolders, name)
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "", map[string]any{
		"path":    folderPath,
		"folders": subFolders,
		"files":   directFiles,
	})
}

// getFolders handles GET /folders?path=<parentFolderPath>
// Returns the immediate sub-folder names under parentFolderPath.
func (s *APIServer) getFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
		return
	}

	ctx := r.Context()
	authUserData, ok := ctx.Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	parentPath := r.URL.Query().Get("path")
	ownerID := sql.NullString{Valid: true, String: authUserData.Id}

	allPaths, err := s.repository.Queries.GetAllFilePathsByOwner(ctx, ownerID)
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	prefix := ""
	if parentPath != "" {
		prefix = parentPath + "/"
	}

	folders := pathutil.ImmediateChildren(prefix, allPaths)

	pkg.WriteJSONResponse(w, http.StatusOK, "", map[string]any{
		"path":    parentPath,
		"folders": folders,
	})
}

// deleteFolder handles DELETE /folders?path=<folderPath>&recursive=true|false
func (s *APIServer) deleteFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
		return
	}

	ctx := r.Context()
	authUserData, ok := ctx.Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	folderPath := r.URL.Query().Get("path")
	if folderPath == "" {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "missing_path", nil)
		return
	}

	recursive := r.URL.Query().Get("recursive")
	if recursive != "true" && recursive != "false" {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "recursive_param_required", "recursive must be 'true' or 'false'")
		return
	}

	ownerID := sql.NullString{Valid: true, String: authUserData.Id}

	// Check if folder has contents
	filesUnder, err := s.repository.Queries.GetFilesByOwnerAndPathPrefix(ctx, sqlc.GetFilesByOwnerAndPathPrefixParams{
		OwnerGoogleID: ownerID,
		FolderPath:    folderPath,
	})
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if len(filesUnder) > 0 && recursive != "true" {
		pkg.WriteJSONResponse(w, http.StatusConflict, "folder_not_empty", "use recursive=true to delete folder and all its contents")
		return
	}

	// Delete all objects from storage
	bucket := fmt.Sprintf("%s-%s", s.bucketHandler.GetBucketBaseName(), authUserData.Id)
	for _, f := range filesUnder {
		if err := s.bucketHandler.DeleteObjectFromBucket(ctx, f.FileName, bucket); err != nil {
			log.Printf("error deleting object %s from storage: %v", f.FileName, err)
		}
	}

	// Delete all DB records under the folder prefix
	deleted, err := s.repository.Queries.DeleteFilesByFolderPrefix(ctx, sqlc.DeleteFilesByFolderPrefixParams{
		OwnerGoogleID: ownerID,
		FolderPath:    folderPath,
	})
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "delete_error", nil)
		return
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "success", map[string]any{
		"folder_deleted": folderPath,
		"files_deleted":  deleted,
	})
}

// moveFile handles POST /files/move
// Body: { "source": "Main/old.txt", "destination": "work/new.txt" }
func (s *APIServer) moveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
		return
	}

	ctx := r.Context()
	authUserData, ok := ctx.Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	type MoveRequest struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}

	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "invalid_json", nil)
		return
	}

	if err := pathutil.Validate(req.Destination); err != nil {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "invalid_destination", err.Error())
		return
	}

	ownerID := sql.NullString{Valid: true, String: authUserData.Id}

	// Check source exists
	_, err := s.repository.Queries.GetFileByOwnerAndName(ctx, sqlc.GetFileByOwnerAndNameParams{
		OwnerGoogleID: ownerID,
		FileName:      req.Source,
	})
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusNotFound, "source_not_found", nil)
		return
	}

	// Check destination is free
	_, dupErr := s.repository.Queries.GetFileByOwnerAndName(ctx, sqlc.GetFileByOwnerAndNameParams{
		OwnerGoogleID: ownerID,
		FileName:      req.Destination,
	})
	if dupErr == nil {
		pkg.WriteJSONResponse(w, http.StatusConflict, "destination_already_exists", nil)
		return
	}

	// Rename object in storage
	bucket := fmt.Sprintf("%s-%s", s.bucketHandler.GetBucketBaseName(), authUserData.Id)
	if err := s.bucketHandler.RenameObject(ctx, bucket, req.Source, bucket, req.Destination); err != nil {
		log.Printf("error renaming object in storage: %v", err)
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "storage_rename_failed", nil)
		return
	}

	// Update DB
	if err := s.repository.Queries.RenameFilePath(ctx, sqlc.RenameFilePathParams{
		OwnerGoogleID: ownerID,
		OldFileName:   req.Source,
		NewFileName:   req.Destination,
	}); err != nil {
		log.Printf("error updating file path in DB: %v", err)
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "db_rename_failed", nil)
		return
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "success", map[string]any{
		"source":      req.Source,
		"destination": req.Destination,
	})
}

// moveFolder handles POST /folders/move
// Body: { "source": "Main/docs", "destination": "work/docs" }
func (s *APIServer) moveFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
		return
	}

	ctx := r.Context()
	authUserData, ok := ctx.Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	type MoveFolderRequest struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}

	var req MoveFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "invalid_json", nil)
		return
	}

	if req.Source == "" || req.Destination == "" {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "source_and_destination_required", nil)
		return
	}

	ownerID := sql.NullString{Valid: true, String: authUserData.Id}

	// Get all files under source folder
	filesToMove, err := s.repository.Queries.GetFilesByOwnerAndPathPrefix(ctx, sqlc.GetFilesByOwnerAndPathPrefixParams{
		OwnerGoogleID: ownerID,
		FolderPath:    req.Source,
	})
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if len(filesToMove) == 0 {
		pkg.WriteJSONResponse(w, http.StatusNotFound, "source_folder_empty_or_not_found", nil)
		return
	}

	// Rename objects in storage
	bucket := fmt.Sprintf("%s-%s", s.bucketHandler.GetBucketBaseName(), authUserData.Id)
	srcPrefix := req.Source + "/"
	dstPrefix := req.Destination + "/"

	for _, f := range filesToMove {
		dstObject := dstPrefix + strings.TrimPrefix(f.FileName, srcPrefix)
		if err := s.bucketHandler.RenameObject(ctx, bucket, f.FileName, bucket, dstObject); err != nil {
			log.Printf("error renaming object %s → %s in storage: %v", f.FileName, dstObject, err)
			pkg.WriteJSONResponse(w, http.StatusInternalServerError, "storage_rename_failed", nil)
			return
		}
	}

	// Batch rename in DB
	if err := s.repository.Queries.RenameFilesByFolderPrefix(ctx, sqlc.RenameFilesByFolderPrefixParams{
		OwnerGoogleID: ownerID,
		OldFolderPath: req.Source,
		NewFolderPath: req.Destination,
	}); err != nil {
		log.Printf("error batch renaming files in DB: %v", err)
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "db_rename_failed", nil)
		return
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "success", map[string]any{
		"source":      req.Source,
		"destination": req.Destination,
		"files_moved": len(filesToMove),
	})
}
