package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/memory/buffer"
	memmanager "GopherDB/internal/core/memory/manager"
	"GopherDB/internal/core/memory/replacer"
	"GopherDB/internal/server"
)

func defaultDataDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "data"
	}
	return filepath.Join(filepath.Dir(exe), "data")
}

func main() {
	port := flag.Int("port", 8081, "Server DB port")
	dataDir := flag.String("dataDir", defaultDataDir(), "Data directory")
	poolSize := flag.Int("poolSize", 256, "Buffer pool size")
	flag.Parse()

	bufferPool, err := buffer.NewHeapBufferPoolManager(
		*poolSize,
		&memmanager.HeapPageFileManager{},
		replacer.NewLRUReplacer(),
		*dataDir,
	)

	if err != nil {
		log.Fatalf("buffer pool init failed: %v", err)
	}

	catalog, err := manager.NewDefaultCatalogManager(*dataDir, bufferPool)
	if err != nil {
		log.Fatalf("catalog init failed: %v", err)
	}

	indexManager := index.NewIndexManager(*dataDir, bufferPool, catalog)
	newServer := server.NewServer(strconv.Itoa(*port), *dataDir, bufferPool, catalog, indexManager)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		newServer.Stop()
	}()

	if err := newServer.Start(); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
