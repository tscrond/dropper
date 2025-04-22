package pkg

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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

func GenerateSecureTokenFromID(id int64) (string, error) {
	// Convert the ID to string
	idStr := strconv.FormatInt(id, 10)

	// Generate 32 random bytes
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", err
	}

	// Mix the ID and the random bytes together
	combined := append([]byte(idStr), randBytes...)

	// Hash the result to create a fixed-length string
	hash := sha256.Sum256(combined)

	// Convert to a 64-character hex string
	return hex.EncodeToString(hash[:]), nil
}
