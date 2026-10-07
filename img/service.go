//go:generate go-enum --sql --marshal --file $GOFILE
package img

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/disintegration/imaging"
	"github.com/dsoprea/go-exif/v3"
	"github.com/marusama/semaphore/v2"

	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

// ErrUnsupportedFormat means the given image format is not supported.
var ErrUnsupportedFormat = errors.New("unsupported image format")

// ErrImageTooLarge means the image is too large to create a thumbnail.
var ErrImageTooLarge = errors.New("image too large for thumbnail generation")

// Maximum dimensions for thumbnail generation to prevent server crashes
const (
	MaxImageWidth  = 10000
	MaxImageHeight = 10000
)

// Service
type Service struct {
	sem semaphore.Semaphore
	// mem limits the total estimated size of images being decoded at once.
	// Nil means no limit.
	mem       semaphore.Semaphore
	memBudget int64
}

type ServiceOption func(*Service)

// WithMemoryBudget limits the memory (in bytes) all concurrent decodes may
// use together. A single image that does not fit into the budget is rejected
// with ErrImageTooLarge. Zero or negative disables the limit.
func WithMemoryBudget(bytes int64) ServiceOption {
	return func(s *Service) {
		if bytes > 0 {
			s.memBudget = bytes
			s.mem = semaphore.New(int(bytes))
		}
	}
}

func New(workers int, options ...ServiceOption) *Service {
	s := &Service{
		sem: semaphore.New(workers),
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// Format is an image file format.
/*
ENUM(
jpeg
png
gif
tiff
bmp
)
*/
type Format int

func (x Format) toImaging() imaging.Format {
	switch x {
	case FormatJpeg:
		return imaging.JPEG
	case FormatPng:
		return imaging.PNG
	case FormatGif:
		return imaging.GIF
	case FormatTiff:
		return imaging.TIFF
	case FormatBmp:
		return imaging.BMP
	default:
		return imaging.JPEG
	}
}

/*
ENUM(
high
medium
low
)
*/
type Quality int

func (x Quality) resampleFilter() imaging.ResampleFilter {
	switch x {
	case QualityHigh:
		return imaging.Lanczos
	case QualityMedium:
		return imaging.Box
	case QualityLow:
		return imaging.NearestNeighbor
	default:
		return imaging.Box
	}
}

/*
ENUM(
fit
fill
)
*/
type ResizeMode int

func (s *Service) FormatFromExtension(ext string) (Format, error) {
	format, err := imaging.FormatFromExtension(ext)
	if err != nil {
		return -1, ErrUnsupportedFormat
	}
	switch format {
	case imaging.JPEG:
		return FormatJpeg, nil
	case imaging.PNG:
		return FormatPng, nil
	case imaging.GIF:
		return FormatGif, nil
	case imaging.TIFF:
		return FormatTiff, nil
	case imaging.BMP:
		return FormatBmp, nil
	}
	return -1, ErrUnsupportedFormat
}

type resizeConfig struct {
	format     Format
	resizeMode ResizeMode
	quality    Quality
}

type Option func(*resizeConfig)

func WithFormat(format Format) Option {
	return func(config *resizeConfig) {
		config.format = format
	}
}

func WithMode(mode ResizeMode) Option {
	return func(config *resizeConfig) {
		config.resizeMode = mode
	}
}

func WithQuality(quality Quality) Option {
	return func(config *resizeConfig) {
		config.quality = quality
	}
}

func (s *Service) Resize(ctx context.Context, in io.Reader, width, height int, out io.Writer, options ...Option) error {
	if err := s.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.sem.Release(1)

	format, imgConfig, wrappedReader, err := s.detectFormat(in)
	if err != nil {
		return err
	}

	config := resizeConfig{
		format:     format,
		resizeMode: ResizeModeFit,
		quality:    QualityMedium,
	}
	for _, option := range options {
		option(&config)
	}

	if config.quality == QualityLow && format == FormatJpeg {
		thm, newWrappedReader, errThm := getEmbeddedThumbnail(wrappedReader)
		wrappedReader = newWrappedReader
		if errThm == nil {
			_, err = out.Write(thm)
			if err == nil {
				return nil
			}
		}
	}

	release, err := s.reserveMemory(ctx, imgConfig)
	if err != nil {
		return err
	}
	defer release()

	img, err := imaging.Decode(wrappedReader, imaging.AutoOrientation(true))
	if err != nil {
		return err
	}

	switch config.resizeMode {
	case ResizeModeFill:
		img = imaging.Fill(img, width, height, imaging.Center, config.quality.resampleFilter())
	case ResizeModeFit:
		fallthrough
	default:
		img = imaging.Fit(img, width, height, config.quality.resampleFilter())
	}

	return imaging.Encode(out, img, config.format.toImaging())
}

func (s *Service) detectFormat(in io.Reader) (Format, image.Config, io.Reader, error) {
	buf := &bytes.Buffer{}
	r := io.TeeReader(in, buf)

	imgConfig, imgFormat, err := image.DecodeConfig(r)
	if err != nil {
		return 0, imgConfig, nil, fmt.Errorf("%s: %w", err.Error(), ErrUnsupportedFormat)
	}

	// Check if image dimensions exceed maximum allowed size
	if imgConfig.Width > MaxImageWidth || imgConfig.Height > MaxImageHeight {
		return 0, imgConfig, nil, fmt.Errorf("image dimensions %dx%d exceed maximum %dx%d: %w",
			imgConfig.Width, imgConfig.Height, MaxImageWidth, MaxImageHeight, ErrImageTooLarge)
	}

	format, err := ParseFormat(imgFormat)
	if err != nil {
		return 0, imgConfig, nil, ErrUnsupportedFormat
	}

	return format, imgConfig, io.MultiReader(buf, in), nil
}

// reserveMemory blocks until the estimated decode cost of the image fits into
// the memory budget. The returned func gives the reservation back.
func (s *Service) reserveMemory(ctx context.Context, cfg image.Config) (func(), error) {
	if s.mem == nil {
		return func() {}, nil
	}

	cost := DecodeCost(cfg)
	if cost > s.memBudget {
		return nil, fmt.Errorf("image %dx%d needs ~%d MiB to decode, budget is %d MiB: %w",
			cfg.Width, cfg.Height, cost>>20, s.memBudget>>20, ErrImageTooLarge)
	}

	if err := s.mem.Acquire(ctx, int(cost)); err != nil {
		return nil, err
	}
	return func() { s.mem.Release(int(cost)) }, nil
}

// DecodeCost estimates how many bytes decoding and resizing the image takes.
// The decoded bitmap is counted twice: auto-orientation and the crop in fill
// mode each make a full-size NRGBA copy of it.
func DecodeCost(cfg image.Config) int64 {
	bytesPerPixel := int64(4)
	switch cfg.ColorModel {
	case color.RGBA64Model, color.NRGBA64Model, color.Gray16Model:
		bytesPerPixel = 8
	}
	return int64(cfg.Width) * int64(cfg.Height) * bytesPerPixel * 2
}

func getEmbeddedThumbnail(in io.Reader) ([]byte, io.Reader, error) {
	buf := &bytes.Buffer{}
	r := io.TeeReader(in, buf)
	wrappedReader := io.MultiReader(buf, in)

	offset := 0
	offsets := []int{12, 30}
	head := make([]byte, 0xffff)

	_, err := r.Read(head)
	if err != nil {
		return nil, wrappedReader, err
	}

	for _, offset = range offsets {
		if _, err = exif.ParseExifHeader(head[offset:]); err == nil {
			break
		}
	}

	if err != nil {
		return nil, wrappedReader, err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, wrappedReader, err
	}

	_, index, err := exif.Collect(im, exif.NewTagIndex(), head[offset:])
	if err != nil {
		return nil, wrappedReader, err
	}

	ifd := index.RootIfd.NextIfd()
	if ifd == nil {
		return nil, wrappedReader, exif.ErrNoThumbnail
	}

	thm, err := ifd.Thumbnail()
	return thm, wrappedReader, err
}
