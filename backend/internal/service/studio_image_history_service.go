package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const studioImageMaxBytes = 32 << 20

var ErrStudioImageAssetNotFound = errors.New("studio image asset not found")

type StudioImageHistoryService struct {
	db          *sql.DB
	storageRoot string
}

type StudioImageHistorySaveImage struct {
	ClientID      string
	B64JSON       string
	RevisedPrompt string
}

type StudioImageHistorySaveInput struct {
	UserID    int64
	GroupID   *int64
	RequestID string
	Model     string
	Size      string
	Prompt    string
	Created   int64
	Images    []StudioImageHistorySaveImage
}

type StudioImageHistoryAsset struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	GroupID       *int64    `json:"group_id,omitempty"`
	RequestID     string    `json:"request_id"`
	ClientID      string    `json:"client_id,omitempty"`
	ImageIndex    int       `json:"image_index"`
	Model         string    `json:"model"`
	Size          string    `json:"size"`
	Prompt        string    `json:"prompt"`
	RevisedPrompt string    `json:"revised_prompt,omitempty"`
	ImageURL      string    `json:"image_url"`
	ContentType   string    `json:"content_type"`
	ByteSize      int64     `json:"byte_size"`
	CreatedAt     time.Time `json:"created_at"`
}

type StudioImageHistoryAssetFile struct {
	Path        string
	ContentType string
}

func NewStudioImageHistoryService(db *sql.DB, cfg *config.Config) *StudioImageHistoryService {
	dataDir := "./data"
	if cfg != nil && strings.TrimSpace(cfg.Pricing.DataDir) != "" {
		dataDir = strings.TrimSpace(cfg.Pricing.DataDir)
	}
	root := filepath.Join(dataDir, "studio-images")
	if absoluteRoot, err := filepath.Abs(root); err == nil {
		root = absoluteRoot
	}
	return &StudioImageHistoryService{db: db, storageRoot: root}
}

func (s *StudioImageHistoryService) Save(ctx context.Context, input StudioImageHistorySaveInput) ([]StudioImageHistoryAsset, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("studio image history is not configured")
	}
	if input.UserID <= 0 {
		return nil, errors.New("user id is required")
	}
	if len(input.Images) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(s.storageRoot, 0755); err != nil {
		return nil, fmt.Errorf("create studio image directory: %w", err)
	}

	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		requestID = fmt.Sprintf("studio-%d-%d", input.UserID, time.Now().UnixNano())
	}
	createdAt := time.Now()
	if input.Created > 0 {
		createdAt = time.Unix(input.Created, 0)
	}

	assets := make([]StudioImageHistoryAsset, 0, len(input.Images))
	for index, image := range input.Images {
		data, err := decodeStudioImageBase64(image.B64JSON)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			continue
		}
		if len(data) > studioImageMaxBytes {
			return nil, fmt.Errorf("studio image is larger than %d bytes", studioImageMaxBytes)
		}

		contentType, extension, err := detectStudioImageContentType(data)
		if err != nil {
			return nil, err
		}
		relativePath := buildStudioImageRelativePath(input.UserID, createdAt, data, index, extension)
		fullPath, err := s.resolveStoragePath(relativePath)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, fmt.Errorf("create studio image asset directory: %w", err)
		}
		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			return nil, fmt.Errorf("write studio image asset: %w", err)
		}

		asset, err := s.upsertAsset(ctx, input, requestID, strings.TrimSpace(image.ClientID), index, relativePath, contentType, int64(len(data)), strings.TrimSpace(image.RevisedPrompt), createdAt)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func (s *StudioImageHistoryService) ListByUser(ctx context.Context, userID int64, limit int) ([]StudioImageHistoryAsset, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("studio image history is not configured")
	}
	if limit <= 0 || limit > 120 {
		limit = 60
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, group_id, request_id, client_id, image_index, model, size, prompt, revised_prompt, content_type, byte_size, created_at
		FROM studio_image_assets
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]StudioImageHistoryAsset, 0, limit)
	for rows.Next() {
		asset, err := scanStudioImageAsset(rows)
		if err != nil {
			return nil, err
		}
		asset.ImageURL = studioImageAssetURL(asset.ID)
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return assets, nil
}

func (s *StudioImageHistoryService) AssetFile(ctx context.Context, userID int64, assetID int64) (*StudioImageHistoryAssetFile, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("studio image history is not configured")
	}

	var storagePath string
	var contentType string
	err := s.db.QueryRowContext(ctx, `
		SELECT storage_path, content_type
		FROM studio_image_assets
		WHERE id = $1 AND user_id = $2
	`, assetID, userID).Scan(&storagePath, &contentType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStudioImageAssetNotFound
	}
	if err != nil {
		return nil, err
	}

	fullPath, err := s.resolveStoragePath(storagePath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrStudioImageAssetNotFound
		}
		return nil, err
	}
	return &StudioImageHistoryAssetFile{Path: fullPath, ContentType: contentType}, nil
}

func (s *StudioImageHistoryService) upsertAsset(
	ctx context.Context,
	input StudioImageHistorySaveInput,
	requestID string,
	clientID string,
	index int,
	relativePath string,
	contentType string,
	byteSize int64,
	revisedPrompt string,
	createdAt time.Time,
) (StudioImageHistoryAsset, error) {
	var asset StudioImageHistoryAsset
	var groupID sql.NullInt64
	if input.GroupID != nil {
		groupID = sql.NullInt64{Int64: *input.GroupID, Valid: true}
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO studio_image_assets (
			user_id, group_id, request_id, client_id, image_index, model, size, prompt, revised_prompt,
			storage_path, content_type, byte_size, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13
		)
		ON CONFLICT (user_id, request_id, image_index) DO UPDATE SET
			client_id = EXCLUDED.client_id,
			model = EXCLUDED.model,
			size = EXCLUDED.size,
			prompt = EXCLUDED.prompt,
			revised_prompt = EXCLUDED.revised_prompt,
			storage_path = EXCLUDED.storage_path,
			content_type = EXCLUDED.content_type,
			byte_size = EXCLUDED.byte_size
		RETURNING id, user_id, group_id, request_id, client_id, image_index, model, size, prompt, revised_prompt, content_type, byte_size, created_at
	`,
		input.UserID,
		groupID,
		requestID,
		clientID,
		index,
		strings.TrimSpace(input.Model),
		strings.TrimSpace(input.Size),
		strings.TrimSpace(input.Prompt),
		revisedPrompt,
		relativePath,
		contentType,
		byteSize,
		createdAt,
	).Scan(
		&asset.ID,
		&asset.UserID,
		&groupID,
		&asset.RequestID,
		&asset.ClientID,
		&asset.ImageIndex,
		&asset.Model,
		&asset.Size,
		&asset.Prompt,
		&asset.RevisedPrompt,
		&asset.ContentType,
		&asset.ByteSize,
		&asset.CreatedAt,
	)
	if err != nil {
		return StudioImageHistoryAsset{}, err
	}
	if groupID.Valid {
		asset.GroupID = &groupID.Int64
	}
	asset.ImageURL = studioImageAssetURL(asset.ID)
	return asset, nil
}

func (s *StudioImageHistoryService) resolveStoragePath(storagePath string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(storagePath))
	if cleaned == "." || cleaned == "" || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("invalid studio image storage path")
	}
	fullPath := filepath.Join(s.storageRoot, cleaned)
	cleanRoot := filepath.Clean(s.storageRoot)
	cleanFull := filepath.Clean(fullPath)
	if cleanFull != cleanRoot && !strings.HasPrefix(cleanFull, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid studio image storage path")
	}
	return cleanFull, nil
}

func decodeStudioImageBase64(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("image data is required")
	}
	if comma := strings.Index(value, ","); comma >= 0 && strings.HasPrefix(strings.ToLower(value[:comma]), "data:") {
		value = value[comma+1:]
	}
	value = strings.TrimSpace(value)
	data, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return data, nil
	}
	data, rawErr := base64.RawStdEncoding.DecodeString(value)
	if rawErr == nil {
		return data, nil
	}
	return nil, err
}

func detectStudioImageContentType(data []byte) (string, string, error) {
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png":
		return contentType, ".png", nil
	case "image/jpeg":
		return contentType, ".jpg", nil
	case "image/webp":
		return contentType, ".webp", nil
	default:
		return "", "", fmt.Errorf("unsupported studio image content type: %s", contentType)
	}
}

func buildStudioImageRelativePath(userID int64, createdAt time.Time, data []byte, index int, extension string) string {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	return filepath.Join(
		fmt.Sprintf("user-%d", userID),
		createdAt.UTC().Format("20060102"),
		fmt.Sprintf("%d-%s-%d%s", createdAt.Unix(), hash[:16], index, extension),
	)
}

func scanStudioImageAsset(scanner interface {
	Scan(dest ...any) error
}) (StudioImageHistoryAsset, error) {
	var asset StudioImageHistoryAsset
	var groupID sql.NullInt64
	if err := scanner.Scan(
		&asset.ID,
		&asset.UserID,
		&groupID,
		&asset.RequestID,
		&asset.ClientID,
		&asset.ImageIndex,
		&asset.Model,
		&asset.Size,
		&asset.Prompt,
		&asset.RevisedPrompt,
		&asset.ContentType,
		&asset.ByteSize,
		&asset.CreatedAt,
	); err != nil {
		return StudioImageHistoryAsset{}, err
	}
	if groupID.Valid {
		asset.GroupID = &groupID.Int64
	}
	return asset, nil
}

func studioImageAssetURL(id int64) string {
	return fmt.Sprintf("/api/v1/studio/images/history/%d/content", id)
}
