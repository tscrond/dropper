package filedata

import "mime/multipart"

type FileData struct {
	MultipartFile  multipart.File
	RequestHeaders *multipart.FileHeader
}

func NewFileData(multipartFile multipart.File, requestHeaders *multipart.FileHeader) *FileData {

	return &FileData{
		MultipartFile:  multipartFile,
		RequestHeaders: requestHeaders,
	}
}
