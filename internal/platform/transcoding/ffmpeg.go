package transcoding

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
)

// TranscodingService handles video transcoding to HLS format
type TranscodingService interface {
	TranscodeToHLS(ctx context.Context, movieID int64, rawFilePath string) (string, error)
}

type transcodingService struct {
	minioClient     *minio.Client
	bucketRaw       string
	bucketProcessed string
	tempDir         string
}

// QualityProfile represents a video quality configuration for HLS
type QualityProfile struct {
	Name       string
	Resolution string
	Bitrate    string
	MaxRate    string
	BufSize    string
}

var (
	// Quality profiles for adaptive bitrate streaming
	qualityProfiles = []QualityProfile{
		{
			Name:       "1080p",
			Resolution: "1920x1080",
			Bitrate:    "5000k",
			MaxRate:    "5350k",
			BufSize:    "7500k",
		},
		{
			Name:       "720p",
			Resolution: "1280x720",
			Bitrate:    "2800k",
			MaxRate:    "2996k",
			BufSize:    "4200k",
		},
		{
			Name:       "480p",
			Resolution: "854x480",
			Bitrate:    "1400k",
			MaxRate:    "1498k",
			BufSize:    "2100k",
		},
		{
			Name:       "360p",
			Resolution: "640x360",
			Bitrate:    "800k",
			MaxRate:    "856k",
			BufSize:    "1200k",
		},
	}
)

// NewTranscodingService creates a new transcoding service
func NewTranscodingService(minioClient *minio.Client, bucketRaw, bucketProcessed string) TranscodingService {
	return &transcodingService{
		minioClient:     minioClient,
		bucketRaw:       bucketRaw,
		bucketProcessed: bucketProcessed,
		tempDir:         "/tmp/transcoding",
	}
}

// TranscodeToHLS transcodes a raw video file to HLS format with multiple quality levels
func (s *transcodingService) TranscodeToHLS(ctx context.Context, movieID int64, rawFilePath string) (string, error) {
	// Create temp directory for transcoding
	workDir := filepath.Join(s.tempDir, fmt.Sprintf("movie-%d", movieID))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir) // Cleanup after transcoding

	// Download raw video from MinIO
	inputPath := filepath.Join(workDir, "input.mp4")
	if err := s.downloadFromMinIO(ctx, rawFilePath, inputPath); err != nil {
		return "", fmt.Errorf("failed to download raw video: %w", err)
	}

	// Create output directory for HLS files
	outputDir := filepath.Join(workDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Transcode to multiple quality levels
	variantPlaylists := []string{}
	for _, profile := range qualityProfiles {
		playlistPath, err := s.transcodeQuality(ctx, inputPath, outputDir, profile)
		if err != nil {
			// Log error but continue with other qualities
			fmt.Printf("Warning: Failed to transcode %s: %v\n", profile.Name, err)
			continue
		}
		variantPlaylists = append(variantPlaylists, playlistPath)
	}

	if len(variantPlaylists) == 0 {
		return "", fmt.Errorf("failed to transcode any quality level")
	}

	// Create master playlist
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")
	if err := s.createMasterPlaylist(masterPlaylistPath, variantPlaylists); err != nil {
		return "", fmt.Errorf("failed to create master playlist: %w", err)
	}

	// Upload all HLS files to MinIO
	hlsBaseURL, err := s.uploadHLSFiles(ctx, movieID, outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to upload HLS files: %w", err)
	}

	return hlsBaseURL, nil
}

// transcodeQuality transcodes video to a specific quality level
func (s *transcodingService) transcodeQuality(ctx context.Context, inputPath, outputDir string, profile QualityProfile) (string, error) {
	// Output playlist name
	playlistName := fmt.Sprintf("%s.m3u8", profile.Name)
	playlistPath := filepath.Join(outputDir, playlistName)
	segmentPattern := filepath.Join(outputDir, fmt.Sprintf("%s_%%03d.ts", profile.Name))

	// Detect available H.264 encoder
	encoder := detectH264Encoder()
	fmt.Printf("Using encoder: %s for %s\n", encoder, profile.Name)

	// Build ffmpeg command based on encoder type
	var args []string

	if encoder == "h264_vaapi" {
		// VAAPI hardware encoding (Intel/AMD)
		// Upload to GPU and convert format properly
		args = []string{
			"-vaapi_device", "/dev/dri/renderD128",
			"-i", inputPath,
			"-vf", fmt.Sprintf("format=nv12,hwupload,scale_vaapi=w=%s:h=%s", getWidth(profile.Resolution), getHeight(profile.Resolution)),
			"-c:v", "h264_vaapi",
			"-b:v", profile.Bitrate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-c:a", "aac",
			"-b:a", "128k",
			"-ac", "2",
			"-f", "hls",
			"-hls_time", "10",
			"-hls_playlist_type", "vod",
			"-hls_segment_type", "mpegts",
			"-hls_segment_filename", segmentPattern,
			playlistPath,
		}
	} else if encoder == "h264_nvenc" {
		// NVIDIA NVENC hardware encoding
		args = []string{
			"-hwaccel", "cuda",
			"-i", inputPath,
			"-vf", fmt.Sprintf("scale=%s", profile.Resolution),
			"-c:v", "h264_nvenc",
			"-preset", "p4", // Medium preset for good quality/speed balance
			"-b:v", profile.Bitrate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-c:a", "aac",
			"-b:a", "128k",
			"-ac", "2",
			"-f", "hls",
			"-hls_time", "10",
			"-hls_playlist_type", "vod",
			"-hls_segment_type", "mpegts",
			"-hls_segment_filename", segmentPattern,
			playlistPath,
		}
	} else {
		// Software encoding fallback (using available encoders)
		args = []string{
			"-i", inputPath,
			"-vf", fmt.Sprintf("scale=%s", profile.Resolution),
			"-c:v", encoder,
		}

		// Add preset/options for specific encoders
		if encoder == "h264" || encoder == "libx264" {
			args = append(args, "-preset", "fast")
		} else if encoder == "libopenh264" {
			// OpenH264 doesn't need extra options - just use default settings
			// The encoder will handle profile automatically
		} else if encoder == "mpeg4" {
			// MPEG-4 specific options
			args = append(args, "-qscale:v", "5") // Good quality for MPEG-4
		}

		args = append(args,
			"-b:v", profile.Bitrate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-c:a", "aac",
			"-b:a", "128k",
			"-ac", "2",
			"-f", "hls",
			"-hls_time", "10",
			"-hls_playlist_type", "vod",
			"-hls_segment_type", "mpegts",
			"-hls_segment_filename", segmentPattern,
			playlistPath,
		)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg command failed: %w", err)
	}

	return playlistName, nil
}

// detectH264Encoder detects the best available H.264 encoder with hardware support verification
func detectH264Encoder() string {
	// Check encoders
	cmd := exec.Command("ffmpeg", "-hide_banner", "-encoders")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Failed to detect encoders, using mpeg4 fallback: %v\n", err)
		return "mpeg4"
	}
	outputStr := string(output)

	// Skip all hardware encoders for now - they're causing issues
	// Intel QSV parameters are incompatible
	// VAAPI processing fails
	// NVENC requires NVIDIA GPU

	fmt.Println("Skipping hardware encoders, using software encoding for compatibility")

	// Use software encoders directly - they work reliably
	swEncoders := []string{"libopenh264", "mpeg4"}
	for _, encoder := range swEncoders {
		if strings.Contains(outputStr, encoder) {
			fmt.Printf("Using software encoder: %s\n", encoder)
			return encoder
		}
	}

	// Ultimate fallback
	fmt.Println("Warning: No preferred encoder found, using mpeg4")
	return "mpeg4"
}

// getWidth extracts width from resolution string (e.g., "1920x1080" -> "1920")
func getWidth(resolution string) string {
	parts := strings.Split(resolution, "x")
	if len(parts) == 2 {
		return parts[0]
	}
	return resolution
}

// getHeight extracts height from resolution string (e.g., "1920x1080" -> "1080")
func getHeight(resolution string) string {
	parts := strings.Split(resolution, "x")
	if len(parts) == 2 {
		return parts[1]
	}
	return resolution
}

// createMasterPlaylist creates an HLS master playlist with all quality variants
func (s *transcodingService) createMasterPlaylist(masterPath string, variantPlaylists []string) error {
	var content strings.Builder
	content.WriteString("#EXTM3U\n")
	content.WriteString("#EXT-X-VERSION:3\n")

	// Add each variant playlist with its metadata
	for i, playlist := range variantPlaylists {
		// Extract quality name from playlist filename (e.g., "1080p.m3u8" -> "1080p")
		qualityName := strings.TrimSuffix(filepath.Base(playlist), ".m3u8")

		// Find matching quality profile
		var profile *QualityProfile
		for j := range qualityProfiles {
			if qualityProfiles[j].Name == qualityName {
				profile = &qualityProfiles[j]
				break
			}
		}

		if profile != nil {
			// Parse resolution
			parts := strings.Split(profile.Resolution, "x")
			if len(parts) == 2 {
				// Parse bitrate (remove 'k' suffix and convert to bits/sec)
				bitrate := strings.TrimSuffix(profile.Bitrate, "k")

				content.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%s000,RESOLUTION=%s\n", bitrate, profile.Resolution))
				content.WriteString(fmt.Sprintf("%s\n", playlist))
			}
		} else {
			// Fallback if profile not found
			content.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d\n", (len(variantPlaylists)-i)*1000000))
			content.WriteString(fmt.Sprintf("%s\n", playlist))
		}
	}

	return os.WriteFile(masterPath, []byte(content.String()), 0644)
}

// downloadFromMinIO downloads a file from MinIO to local filesystem
func (s *transcodingService) downloadFromMinIO(ctx context.Context, objectName, destPath string) error {
	// Get object from MinIO
	object, err := s.minioClient.GetObject(ctx, s.bucketRaw, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer object.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy object content to file
	if _, err := destFile.ReadFrom(object); err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}

// uploadHLSFiles uploads all HLS files from output directory to MinIO
func (s *transcodingService) uploadHLSFiles(ctx context.Context, movieID int64, outputDir string) (string, error) {
	// Base path in MinIO for this movie's HLS files
	basePath := fmt.Sprintf("movie-%d", movieID)

	// Walk through output directory and upload all files
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		// MinIO object name
		objectName := filepath.Join(basePath, relPath)

		// Determine content type
		contentType := "application/octet-stream"
		if strings.HasSuffix(path, ".m3u8") {
			contentType = "application/vnd.apple.mpegurl"
		} else if strings.HasSuffix(path, ".ts") {
			contentType = "video/mp2t"
		}

		// Upload file to MinIO
		_, err = s.minioClient.FPutObject(ctx, s.bucketProcessed, objectName, path, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", objectName, err)
		}

		fmt.Printf("Uploaded: %s\n", objectName)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload HLS files: %w", err)
	}

	// Return URL to master playlist
	masterPlaylistURL := fmt.Sprintf("%s/master.m3u8", basePath)
	return masterPlaylistURL, nil
}
