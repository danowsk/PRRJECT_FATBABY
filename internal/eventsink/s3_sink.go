package eventsink

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

type S3Uploader interface {
	PutObject(ctx context.Context, bucket, key string, body []byte, contentType string) error
}

type S3Sink struct {
	Uploader S3Uploader
	Bucket   string
}

func (s S3Sink) Write(ctx context.Context, evt eventstore.Event) error {
	key := BuildS3ObjectKey(evt)
	body, err := json.Marshal(evt)
	if err != nil { return fmt.Errorf("marshal event: %w", err) }
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil { return err }
	if err := zw.Close(); err != nil { return err }
	return s.Uploader.PutObject(ctx, s.Bucket, key, buf.Bytes(), "application/json")
}

var unsafeRe = regexp.MustCompile(`[^a-z0-9_-]+`)

func BuildS3ObjectKey(evt eventstore.Event) string {
	ts := evt.OccurredAt.UTC()
	if ts.IsZero() { ts = evt.IngestedAt.UTC() }
	if ts.IsZero() { ts = time.Now().UTC() }
	source := strings.ToLower(strings.TrimSpace(evt.Source))
	source = unsafeRe.ReplaceAllString(source, "_")
	source = strings.Trim(source, "_")
	if source == "" { source = "unknown" }
	return fmt.Sprintf("source=%s/year=%04d/month=%02d/day=%02d/hour=%02d/%s.json.zst", source, ts.Year(), ts.Month(), ts.Day(), ts.Hour(), evt.ID)
}
