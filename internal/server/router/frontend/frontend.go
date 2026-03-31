package frontend

import (
	"context"
	"embed"
	"io/fs"
	"net/http"

	"github.com/gorilla/mux"
)

// 注意这里要相对路径（当前文件的相对路径）
//
//go:generate go run ./generate/mian.go -srcDir ./dist -originalText http://proxy.51d.ink -newText http://localhost:8888
//go:embed dist/*
var embeddedFiles embed.FS

type FrontendService struct {
	Path string
}

func NewFrontendService(path string) *FrontendService {
	return &FrontendService{
		Path: "/" + path + "/",
	}
}

func (fe *FrontendService) Serve(ctx context.Context, r *mux.Router) error {
	// 这个是嵌入
	prefix := fe.Path
	r.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.FileServer(getFileSystem("dist"))))
	// // 这个f需要目录
	return nil
}

// Register healthz endpoint.

func getFileSystem(path string) http.FileSystem {
	fs, err := fs.Sub(embeddedFiles, path)
	if err != nil {
		panic(err)
	}
	return http.FS(fs)
}
