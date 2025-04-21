package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

func GetUserBucketName(bucketBaseName, userID string) string {
	return fmt.Sprintf("%s-%s", bucketBaseName, userID)
}

func LoadServiceAccount(path string) (email string, privateKey []byte, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}

	var creds struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", nil, err
	}

	if creds.ClientEmail == "" || creds.PrivateKey == "" {
		return "", nil, fmt.Errorf("invalid service account file: missing email or private key")
	}

	return creds.ClientEmail, []byte(creds.PrivateKey), nil
}

func RandToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
