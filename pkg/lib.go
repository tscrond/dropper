package pkg

import "fmt"

func GetUserBucketName(bucketBaseName, userID string) string {
	return fmt.Sprintf("%s-%s", bucketBaseName, userID)
}
