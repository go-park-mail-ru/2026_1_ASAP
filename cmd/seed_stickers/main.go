package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
)

type manifest struct {
	Name      string            `json:"name"`
	Title     string            `json:"title"`
	Thumbnail string            `json:"thumbnail"`
	Stickers  []manifestSticker `json:"stickers"`
}

type manifestSticker struct {
	File  string `json:"file"`
	Slug  string `json:"slug"`
	Emoji string `json:"emoji"`
}

type stickerSeed struct {
	FileURL   string
	FilePath  string
	ObjectKey string
	Slug      string
	Emoji     *string
	Width     int
	Height    int
	SortOrder int
}

var sortPrefixRE = regexp.MustCompile(`^([0-9]+)[_-](.+)$`)

func main() {
	assetsRoot := flag.String("assets", "assets/stickers", "stickers assets root")
	packDirName := flag.String("pack", "", "single pack directory name to seed")
	chatConfigPath := flag.String("chat-config", "", "chat config path")
	mediaConfigPath := flag.String("media-config", "", "media config path")
	flag.Parse()

	ctx := context.Background()
	chatCfg, err := loadChatConfig(*chatConfigPath)
	if err != nil {
		log.Fatalf("load chat config: %v", err)
	}
	mediaCfg, err := loadMediaConfig(*mediaConfigPath)
	if err != nil {
		log.Fatalf("load media config: %v", err)
	}

	db, err := newPostgres(ctx, chatCfg.PostgresConfig)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	client, err := newMinio(ctx, mediaCfg.S3Config)
	if err != nil {
		log.Fatalf("connect minio: %v", err)
	}

	packDirs, err := discoverPackDirs(*assetsRoot, *packDirName)
	if err != nil {
		log.Fatalf("discover sticker packs: %v", err)
	}
	for _, packDir := range packDirs {
		if err := seedPack(ctx, db, client, mediaCfg.S3Config, packDir); err != nil {
			log.Fatalf("seed pack %s: %v", packDir, err)
		}
	}
}

func loadChatConfig(path string) (*config.ChatConfig, error) {
	if strings.TrimSpace(path) == "" {
		return config.LoadChatConfig()
	}
	return config.LoadChatConfigFromPath(path)
}

func loadMediaConfig(path string) (*config.MediaConfig, error) {
	if strings.TrimSpace(path) == "" {
		return config.LoadMediaConfig()
	}
	return config.LoadMediaConfigFromPath(path)
}

func newPostgres(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return pgxpool.New(ctx, connStr)
}

func newMinio(ctx context.Context, cfg config.S3Config) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	if err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
		exists, existsErr := client.BucketExists(ctx, cfg.Bucket)
		if existsErr != nil || !exists {
			return nil, fmt.Errorf("bucket %s: %w", cfg.Bucket, err)
		}
	}
	policy := `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": ["s3:GetObject"],
            "Effect": "Allow",
            "Principal": {"AWS": ["*"]},
            "Resource": ["arn:aws:s3:::` + cfg.Bucket + `/*"],
            "Sid": ""
        }
    ]
	}`
	if err = client.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
		return nil, fmt.Errorf("set bucket policy: %w", err)
	}
	return client, nil
}

func discoverPackDirs(root, pack string) ([]string, error) {
	if pack != "" {
		return []string{filepath.Join(root, pack)}, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(root, entry.Name()))
		}
	}
	return dirs, nil
}

func seedPack(ctx context.Context, db *pgxpool.Pool, client *minio.Client, s3 config.S3Config, packDir string) error {
	mf, err := readManifest(packDir)
	if err != nil {
		return err
	}
	packSlug := slugify(mf.Name)
	if packSlug == "" {
		packSlug = slugify(filepath.Base(packDir))
	}
	if packSlug == "" {
		return fmt.Errorf("pack slug is empty")
	}
	title := strings.TrimSpace(mf.Title)
	if title == "" {
		title = mf.Name
	}

	stickers := make([]stickerSeed, 0, len(mf.Stickers))
	for i, item := range mf.Stickers {
		seed, err := prepareSticker(packDir, packSlug, item, i)
		if err != nil {
			return err
		}
		stickers = append(stickers, seed)
	}

	for i := range stickers {
		fileURL, err := uploadSticker(ctx, client, s3, stickers[i])
		if err != nil {
			return err
		}
		stickers[i].FileURL = fileURL
	}

	thumbnailURL := ""
	if mf.Thumbnail != "" {
		for _, sticker := range stickers {
			if filepath.Base(sticker.FilePath) == mf.Thumbnail {
				thumbnailURL = sticker.FileURL
				break
			}
		}
	}
	if thumbnailURL == "" && len(stickers) > 0 {
		thumbnailURL = stickers[0].FileURL
	}

	packID, err := upsertPack(ctx, db, mf.Name, packSlug, title, thumbnailURL)
	if err != nil {
		return err
	}
	for _, sticker := range stickers {
		if err := upsertSticker(ctx, db, packID, sticker); err != nil {
			return err
		}
	}

	log.Printf("seeded sticker pack %q: %d stickers", mf.Name, len(stickers))
	return nil
}

func readManifest(packDir string) (*manifest, error) {
	data, err := os.ReadFile(filepath.Join(packDir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var mf manifest
	if err = json.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if strings.TrimSpace(mf.Name) == "" {
		return nil, fmt.Errorf("manifest name is required")
	}
	if len(mf.Stickers) == 0 {
		return nil, fmt.Errorf("manifest stickers is empty")
	}
	return &mf, nil
}

func prepareSticker(packDir, packSlug string, item manifestSticker, idx int) (stickerSeed, error) {
	if strings.TrimSpace(item.File) == "" {
		return stickerSeed{}, fmt.Errorf("sticker file is required at index %d", idx)
	}
	filePath := filepath.Join(packDir, item.File)
	width, height, err := imageSize(filePath)
	if err != nil {
		return stickerSeed{}, err
	}
	sortOrder, slug := sortAndSlug(item.File, idx)
	if item.Slug != "" {
		slug = slugify(item.Slug)
	}
	if slug == "" {
		return stickerSeed{}, fmt.Errorf("sticker slug is empty for %s", item.File)
	}
	objectKey := fmt.Sprintf("stickers/%s/%s%s", packSlug, slug, strings.ToLower(filepath.Ext(item.File)))

	var emoji *string
	if strings.TrimSpace(item.Emoji) != "" {
		v := strings.TrimSpace(item.Emoji)
		emoji = &v
	}

	return stickerSeed{
		FilePath:  filePath,
		ObjectKey: objectKey,
		Slug:      slug,
		Emoji:     emoji,
		Width:     width,
		Height:    height,
		SortOrder: sortOrder,
	}, nil
}

func imageSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open image %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("decode image size %s: %w", path, err)
	}
	return cfg.Width, cfg.Height, nil
}

func uploadSticker(ctx context.Context, client *minio.Client, cfg config.S3Config, sticker stickerSeed) (string, error) {
	f, err := os.Open(sticker.FilePath)
	if err != nil {
		return "", fmt.Errorf("open sticker %s: %w", sticker.FilePath, err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat sticker %s: %w", sticker.FilePath, err)
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(sticker.FilePath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = client.PutObject(ctx, cfg.Bucket, sticker.ObjectKey, f, info.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload sticker %s: %w", sticker.FilePath, err)
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(cfg.PublicURL(), "/"), cfg.Bucket, sticker.ObjectKey), nil
}

func upsertPack(ctx context.Context, db *pgxpool.Pool, name, slug, title, thumbnailURL string) (int64, error) {
	const q = `
INSERT INTO sticker_packs (name, slug, title, thumbnail_url, sort_order)
VALUES ($1, $2, $3, $4, 0)
ON CONFLICT (slug) WHERE slug IS NOT NULL
DO UPDATE SET
    name = EXCLUDED.name,
    title = EXCLUDED.title,
    thumbnail_url = EXCLUDED.thumbnail_url,
    updated_at = now()
RETURNING id`
	var id int64
	if err := db.QueryRow(ctx, q, name, slug, title, thumbnailURL).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert sticker pack: %w", err)
	}
	return id, nil
}

func upsertSticker(ctx context.Context, db *pgxpool.Pool, packID int64, sticker stickerSeed) error {
	const q = `
INSERT INTO stickers (pack_id, file_url, slug, emoji, width, height, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (pack_id, slug) WHERE slug IS NOT NULL
DO UPDATE SET
    file_url = EXCLUDED.file_url,
    emoji = EXCLUDED.emoji,
    width = EXCLUDED.width,
    height = EXCLUDED.height,
    sort_order = EXCLUDED.sort_order,
    updated_at = now()`
	if _, err := db.Exec(ctx, q, packID, sticker.FileURL, sticker.Slug, sticker.Emoji, sticker.Width, sticker.Height, sticker.SortOrder); err != nil {
		return fmt.Errorf("upsert sticker %s: %w", sticker.Slug, err)
	}
	return nil
}

func sortAndSlug(file string, idx int) (int, string) {
	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	sortOrder := idx + 1
	slugPart := base
	matches := sortPrefixRE.FindStringSubmatch(base)
	if len(matches) == 3 {
		if parsed, err := strconv.Atoi(matches[1]); err == nil {
			sortOrder = parsed
		}
		slugPart = matches[2]
	}
	return sortOrder, slugify(slugPart)
}

func slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
