# S3 Cloud Storage Driver — Implementation Plan

## Overview

Add an AWS S3 storage driver that implements the `ObjectStorage` interface, selectable at runtime via a `STORAGE_PROVIDER` env var. Uses the AWS SDK for Go v2 with the default credential chain and a single-bucket-with-prefixes strategy for scalability.

---

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Auth method | AWS default credential chain | Cloud-native: supports EKS IRSA, EC2 instance profiles, env vars, `~/.aws/credentials` — no custom key file needed |
| Bucket strategy | Single bucket + per-user key prefixes (`<userId>/filename`) | S3 bucket names are globally unique with a ~100/account soft limit; prefixes scale infinitely and are the S3 best practice |
| Region | Configurable via `AWS_REGION` env var | Standard AWS convention, picked up automatically by the SDK |
| Provider selection | `STORAGE_PROVIDER` env var (`gcs` or `s3`) | Replaces hardcoded `"gcs"` in `main.go` |
| S3-compatible endpoints | Not in scope (S3 only) | Can be added later with an optional `S3_ENDPOINT` env var |

---

## Architecture

### Interface (no changes)

The existing `ObjectStorage` interface in `internal/cloud_storage/types/object.go` stays as-is:

```
SendFileToBucket(ctx, *FileData) error
BucketExists(ctx, fullBucketName) (bool, error)
CreateBucketIfNotExists(ctx, userId) error
GetUserBucketData(ctx, id) (any, error)
GetUserBucketName(ctx) (string, error)
GetBucketBaseName() string
GenerateSignedURL(ctx, bucket, object, expiresAt) (string, error)
DeleteObjectFromBucket(ctx, object, bucket) error
DeleteBucket(ctx, bucket) error
Close() error
```

### Semantic mapping: GCS concepts → S3 equivalents

| GCS (current) | S3 (new driver) |
|---|---|
| Per-user bucket `<base>-<userId>` | Single bucket, objects keyed as `<userId>/<filename>` |
| `storage.NewClient` + service account JSON | `s3.NewFromConfig(cfg)` via default credential chain |
| `Bucket.Create()` | No-op or `CreateBucket` (bucket already exists; "user bucket" is just a prefix) |
| `Bucket.Attrs()` | `HeadBucket` for existence; synthetic attrs from `ListObjectsV2` for user data |
| `Object.NewWriter` + `io.Copy` | `s3.PutObject` with the `io.Reader` body |
| `Object.Attrs()` | `s3.HeadObject` to get content type, size, MD5/ETag |
| `Bucket.Objects(ctx, nil)` iteration | `s3.ListObjectsV2` with `Prefix: userId + "/"` |
| `storage.SignedURL()` with private key | `s3.PresignClient.PresignGetObject()` — no private key needed, uses SDK credentials |
| `Object.Delete()` | `s3.DeleteObject` |
| `Bucket.Delete()` — deletes all objects first | `s3.DeleteObjects` (batch) then optionally prefix cleanup; no actual bucket deletion since it's shared |
| `Client.Close()` | No-op (S3 client has no `Close`) |

---

## Environment Variables

New env vars required when `STORAGE_PROVIDER=s3`:

| Variable | Required | Description |
|---|---|---|
| `STORAGE_PROVIDER` | Yes | `gcs` or `s3` (replaces hardcoded value) |
| `AWS_REGION` | Yes | AWS region, e.g. `eu-west-1` |
| `S3_BUCKET_NAME` | Yes | The single S3 bucket name |
| `AWS_ACCESS_KEY_ID` | No* | Static creds (auto-detected by SDK) |
| `AWS_SECRET_ACCESS_KEY` | No* | Static creds (auto-detected by SDK) |

\* Not needed when running on EKS with IRSA or EC2 with an instance profile.

---

## Files to Create

### 1. `internal/cloud_storage/s3/s3_handler.go`

New S3 driver implementing `ObjectStorage`. Core structure:

```go
type S3BucketHandler struct {
    repository     *repo.Repository
    Client         *s3.Client
    PresignClient  *s3.PresignClient
    BaseBucketName string
    Region         string
}
```

**Method implementations:**

| Method | S3 Implementation |
|---|---|
| `NewS3BucketHandler(bucket, region, repo)` | `config.LoadDefaultConfig` → `s3.NewFromConfig` + `s3.NewPresignClient` |
| `SendFileToBucket` | `s3.PutObject` with `Key: <userId>/<filename>`, content-type detection same as GCS driver, then DB insert (same logic) |
| `BucketExists` | `s3.HeadBucket` — checks if the shared bucket exists |
| `CreateBucketIfNotExists` | `s3.HeadBucket` → `s3.CreateBucket` if not found (normally a one-time setup; user prefixes don't need creation) |
| `GetUserBucketData` | `s3.ListObjectsV2` with `Prefix: userId + "/"` → build `mappings.BucketData` with synthetic bucket attrs |
| `GetUserBucketName` | Return `baseBucketName` (the shared bucket; user isolation is via prefix) |
| `GetBucketBaseName` | Return `baseBucketName` |
| `GenerateSignedURL` | `PresignClient.PresignGetObject` with expiry duration — no service account file needed |
| `DeleteObjectFromBucket` | `s3.DeleteObject` with `Key: object` (full prefixed key) |
| `DeleteBucket` | `s3.ListObjectsV2` prefix → `s3.DeleteObjects` batch delete of all user objects — does NOT delete the shared bucket |
| `Close` | No-op, return `nil` |

---

## Files to Modify

### 2. `internal/cloud_storage/factory/factory.go`

Add `case "s3"` that reads `S3_BUCKET_NAME` and `AWS_REGION` from env, calls `s3handler.NewS3BucketHandler(...)`.

### 3. `cmd/main.go`

- Change `storageProvider := "gcs"` → `storageProvider := os.Getenv("STORAGE_PROVIDER")` with a fallback default to `"gcs"`.

### 4. `internal/api/data_delete.go`

The `deleteFile` handler constructs the bucket object key as:
```go
bucket := fmt.Sprintf("%s-%s", s.bucketHandler.GetBucketBaseName(), authUserData.Id)
```
This pattern assumes per-user buckets. For S3, the object key is `<userId>/<filename>`, and the bucket is the shared base bucket.

**Approach:** The `deleteFile` and `deleteAccount` handlers build bucket names outside the driver, which breaks abstraction. Two options:

- **Option A (minimal change):** In the S3 driver, `GetBucketBaseName()` returns the shared bucket name. The constructed `bucket` var (`<base>-<userId>`) won't match, so `DeleteObjectFromBucket` must interpret the object key correctly. This is fragile.
- **Option B (clean):** Change `deleteFile` to use a helper or let the driver handle prefixing internally. Since the GCS driver already receives `object` and `bucket` separately, and the S3 driver knows the bucket is always `baseBucketName`, the S3 `DeleteObjectFromBucket` can **ignore the `bucket` param** and use `baseBucketName` while prepending the userId prefix to the object key — but it doesn't have the userId in that call.

**Chosen approach:** The S3 driver will store objects with the prefixed key (`<userId>/<filename>`) in the DB and bucket. The `object` parameter to `DeleteObjectFromBucket` will already be the full key (including prefix). The `bucket` parameter will be interpreted by each driver as appropriate:
- GCS: uses `bucket` as the actual bucket name
- S3: uses its own `baseBucketName` (ignores the `bucket` param since there's only one bucket)

The `deleteFile` handler already passes the filename as `object`, but the filename in the DB doesn't have the prefix. So the S3 driver's `DeleteObjectFromBucket` will need to reconstruct the full key. Since it receives `bucket` which is `<base>-<userId>`, it can extract the userId from it.

**Cleaner alternative selected:** Store the full prefixed key (`<userId>/<filename>`) as the `file_name` in the DB when using S3. This way all lookups, shares, and deletes use the full key consistently. However, this changes the display name. 

**Final decision:** The S3 `DeleteObjectFromBucket(ctx, object, bucket)` implementation will:
- Accept the same `bucket` format as GCS (`<baseName>-<userId>`)
- Internally use `baseBucketName` as the real S3 bucket
- Construct the S3 key as `<userId>/<object>` by extracting the userId suffix from the passed `bucket` string

This requires **zero changes** to `data_delete.go` or any other handler. Same approach for `DeleteBucket` — it receives the composed name, extracts the userId, lists objects by prefix, and batch-deletes.

### 5. `pkg/lib.go`

Add a helper `ExtractUserIdFromBucketName(baseName, compositeName) string` to parse the userId from `<baseName>-<userId>`. Used by the S3 driver to maintain compatibility with the existing handler calling convention.

### 6. `go.mod` / `go.sum`

Add `github.com/aws/aws-sdk-go-v2/service/s3` and `github.com/aws/aws-sdk-go-v2/feature/s3/manager` (if needed for multipart uploads). Some AWS SDK v2 modules are already in go.mod for SES.

---

## Key Implementation Details

### Presigned URLs (GenerateSignedURL)

GCS requires a service account private key to sign URLs. S3 uses the presign client which signs with whatever credentials the SDK loaded:

```go
presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
    Bucket: aws.String(bucketName),
    Key:    aws.String(objectKey),
}, s3.WithPresignExpires(time.Until(expiresAt)))
```

The `bucket` and `object` params from the callers in `sharing_get.go` come from the DB (`user_bucket` and `file_name` columns). For S3:
- The `bucket` field in DB will store the shared bucket name
- The `object` will be the plain filename
- The S3 driver's `GenerateSignedURL` will construct the full key by extracting info from the bucket param or using a convention

**Resolution:** When the S3 driver stores a file via `SendFileToBucket`, it stores:
- `user_bucket` in DB → the base S3 bucket name (from `GetUserBucketName`)  
- `file_name` in DB → just the filename (no prefix)

Then in `GenerateSignedURL(ctx, bucket, object, expiresAt)`, the S3 driver must know the userId to build the S3 key. Since `bucket` from DB already tells us which "user bucket" it is:
- For GCS it's `<base>-<userId>` → used as literal bucket name
- For S3 the driver needs to look up the userId from DB or embed it differently

**Simplest solution:** In the S3 driver, `GetUserBucketName(ctx)` returns `<baseBucketName>-<userId>` (same format as GCS). The S3 driver then parses this to extract the userId when needed. This keeps DB values and all handler code 100% identical between drivers.

### Object Metadata (SendFileToBucket)

After `PutObject`, call `HeadObject` to get the ETag (MD5 for non-multipart uploads), size, and content type for the DB insert — mirroring what the GCS driver gets from `Object.Attrs()`.

### User Bucket Data (GetUserBucketData)

Build a `mappings.BucketData` struct from `ListObjectsV2` results with `Prefix: userId + "/"`. Populate synthetic bucket-level attributes (name, storage class, creation time) since S3 doesn't have per-prefix metadata.

---

## Dependency Additions

```
github.com/aws/aws-sdk-go-v2/service/s3
```

Already in go.mod (transitive/direct):
```
github.com/aws/aws-sdk-go-v2
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/credentials
```

---

## Implementation Order

1. Add `S3_BUCKET_NAME`, `AWS_REGION` env vars to `docker-compose.yaml` (optional, for local dev)
2. Add `ExtractUserIdFromBucketName` helper to `pkg/lib.go`
3. Create `internal/cloud_storage/s3/s3_handler.go` — full `ObjectStorage` implementation
4. Update `internal/cloud_storage/factory/factory.go` — add `case "s3"`
5. Update `cmd/main.go` — make `storageProvider` configurable via `STORAGE_PROVIDER` env var
6. Run `go mod tidy` to pull S3 SDK dependency
7. Build and verify compilation
8. Manual testing with a real S3 bucket or LocalStack

---

## Testing Strategy

- **Unit**: Mock the S3 client interface to test each method's logic (key construction, error handling)
- **Integration**: Use LocalStack or a real S3 bucket with `STORAGE_PROVIDER=s3`
- **Regression**: Ensure GCS path still works with `STORAGE_PROVIDER=gcs`

---

## Risk & Edge Cases

| Risk | Mitigation |
|---|---|
| S3 ETag ≠ MD5 for multipart uploads | Use `PutObject` (not multipart manager) for files; ETag = MD5 for single-part uploads. Can add `Content-MD5` header for verification. |
| Bucket name collision in handler params | S3 driver always uses `baseBucketName` internally, parses userId from the `<base>-<userId>` format passed by handlers |
| Presigned URL auth differences | S3 presign client handles all auth via the default chain; no service account file parsing needed |
| `DeleteBucket` called for account deletion | S3 driver deletes all objects under the user's prefix — does NOT delete the shared bucket |
| `Close()` called on shutdown | S3 client has no `Close()`; return nil |
