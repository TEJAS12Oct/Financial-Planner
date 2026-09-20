package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrSignedReportURLRequired = errors.New("signed report URL is required")
	ErrInvalidReportURLExpiry  = errors.New("invalid report URL expiry")
	ErrReportURLExpired        = errors.New("report URL has expired; request a new report link")
	ErrInvalidReportSignature  = errors.New("invalid report URL signature")
)

type ReportSigner struct {
	enabled bool
	secret  []byte
	ttl     time.Duration
}

func NewReportSigner(cfg Config) *ReportSigner {
	return &ReportSigner{
		enabled: cfg.Enabled,
		secret:  []byte(cfg.ReportSigningSecret),
		ttl:     cfg.ReportLinkTTL,
	}
}

func (s *ReportSigner) Enabled() bool {
	return s != nil && s.enabled
}

func (s *ReportSigner) SignURL(rawPath string) string {
	if s == nil || !s.enabled || strings.TrimSpace(rawPath) == "" {
		return rawPath
	}

	canonicalPath := CanonicalReportPath(rawPath)
	expires := time.Now().Add(s.ttl).Unix()
	signature := s.signature(canonicalPath, expires)

	separator := "?"
	if strings.Contains(canonicalPath, "?") {
		separator = "&"
	}

	return canonicalPath + separator +
		"expires=" + strconv.FormatInt(expires, 10) +
		"&signature=" + url.QueryEscape(signature)
}

func (s *ReportSigner) Validate(rawPath, expiresText, signature string) error {
	if s == nil || !s.enabled {
		return nil
	}
	if strings.TrimSpace(expiresText) == "" || strings.TrimSpace(signature) == "" {
		return ErrSignedReportURLRequired
	}

	expires, err := strconv.ParseInt(expiresText, 10, 64)
	if err != nil {
		return ErrInvalidReportURLExpiry
	}
	if time.Now().Unix() > expires {
		return ErrReportURLExpired
	}

	canonicalPath := CanonicalReportPath(rawPath)
	expected := s.signature(canonicalPath, expires)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidReportSignature
	}

	return nil
}

func (s *ReportSigner) signature(path string, expires int64) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(path))
	mac.Write([]byte("\n"))
	mac.Write([]byte(strconv.FormatInt(expires, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func CanonicalReportPath(rawPath string) string {
	path := strings.TrimSpace(rawPath)
	if parsed, err := url.Parse(path); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
