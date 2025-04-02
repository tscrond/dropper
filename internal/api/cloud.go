package api

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/tscrond/dropper/internal/filedata"
)

func (s *APIServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSON(w, map[string]interface{}{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}
	ctx := context.Background()

	// Get file from request
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to parse file from request", http.StatusBadRequest)
		log.Println(err)
		return
	}
	defer file.Close()

	// Create fileData object
	fileData := filedata.NewFileData(file, header)
	if fileData == nil {
		http.Error(w, "Invalid file data", http.StatusInternalServerError)
		log.Println("Invalid file data")
		return
	}

	if err := s.bucketHandler.SendFileToBucket(ctx, fileData); err != nil {
		http.Error(w, "Failed to send file to bucket", http.StatusInternalServerError)
		log.Fatal(err)
	}

	fmt.Fprintf(w, "Files uploaded successfully: %+v\n", fileData.RequestHeaders.Filename)
}
