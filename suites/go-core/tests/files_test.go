//go:build e2e && svc_files

package tests

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	filesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/files/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

func TestFilesGetFileMetadataRequiresID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := newFilesClient(t)
	_, err := client.GetFileMetadata(ctx, &filesv1.GetFileMetadataRequest{})
	requireFilesGRPCCode(t, err, codes.InvalidArgument)
}

func TestFilesGetFileContentRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := newFilesClient(t)
	content := []byte("hello world")

	uploadStream, err := client.UploadFile(ctx)
	if err != nil {
		t.Fatalf("start upload: %v", err)
	}
	metadata := &filesv1.UploadFileRequest{
		Payload: &filesv1.UploadFileRequest_Metadata{
			Metadata: &filesv1.UploadFileMetadata{
				Filename:    "hello.txt",
				ContentType: "text/plain",
				SizeBytes:   int64(len(content)),
			},
		},
	}
	if err := uploadStream.Send(metadata); err != nil {
		t.Fatalf("send metadata: %v", err)
	}
	if err := uploadStream.Send(&filesv1.UploadFileRequest{
		Payload: &filesv1.UploadFileRequest_Chunk{
			Chunk: &filesv1.UploadFileChunk{Data: content[:5]},
		},
	}); err != nil {
		t.Fatalf("send chunk: %v", err)
	}
	if err := uploadStream.Send(&filesv1.UploadFileRequest{
		Payload: &filesv1.UploadFileRequest_Chunk{
			Chunk: &filesv1.UploadFileChunk{Data: content[5:]},
		},
	}); err != nil {
		t.Fatalf("send chunk: %v", err)
	}

	uploadResp, err := uploadStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("finish upload: %v", err)
	}
	if uploadResp.GetFile() == nil || uploadResp.GetFile().GetId() == "" {
		t.Fatalf("expected file id in upload response")
	}

	downloadStream, err := client.GetFileContent(ctx, &filesv1.GetFileContentRequest{FileId: uploadResp.File.Id})
	if err != nil {
		t.Fatalf("get file content: %v", err)
	}
	var received []byte
	for {
		resp, err := downloadStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("receive chunk: %v", err)
		}
		received = append(received, resp.GetChunkData()...)
	}
	if !bytes.Equal(received, content) {
		t.Fatalf("downloaded content does not match")
	}
}

func TestFilesGetFileContentNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := newFilesClient(t)
	stream, err := client.GetFileContent(ctx, &filesv1.GetFileContentRequest{FileId: uuid.NewString()})
	if err == nil {
		_, err = stream.Recv()
	}
	requireFilesGRPCCode(t, err, codes.NotFound)
}
