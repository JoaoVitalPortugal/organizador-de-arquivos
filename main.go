package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify" // Caminho correto sem o ://
	"github.com/gen2brain/beeep"
)

var (
	targetDir string
	debug     bool

	defaultDir = filepath.Join(os.Getenv("HOME"), "Downloads")

	defaultImagesDir = "Images"
	defaultVideoDir  = "Vídeos"
	defaultAudioDir  = "Músicas"

	suffixDirMapping = map[string]string{
		// For images
		"jpeg": defaultImagesDir, "jpg": defaultImagesDir, "png": defaultImagesDir,
		// For audios
		"mp4": defaultVideoDir, "mkv": defaultVideoDir, "mov": defaultVideoDir,
		// For videos
		"mp3": defaultAudioDir, "wav": defaultAudioDir, "aac": defaultAudioDir,
	}
)

func main() {
	flag.StringVar(&targetDir, "target-dir", defaultDir, `Directory from which the app should watch events.`)
	flag.StringVar(&defaultImagesDir, "images-dir", defaultImagesDir, `Directory where we move image files.`)
	flag.StringVar(&defaultAudioDir, "audios-dir", defaultAudioDir, `Directory where we move audio files.`)
	flag.StringVar(&defaultVideoDir, "videos-dir", defaultVideoDir, `Directory where we move video files.`)
	flag.BoolVar(&debug, "debug", false, `Enable debuging logging. By default is disabled`)
	flag.Parse()

	handlerOpts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if debug {
		handlerOpts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, handlerOpts))
	defer func() {
		if err := recover(); err != nil {
			logger.Error("Aconteceu algum error irrecuperável", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.ErrorContext(ctx, "não foi possível criar um 'watcher' para o path", "target-dir", targetDir)
	}
	defer watcher.Close()
	
	err = watcher.Add(targetDir)
	if err != nil {
		logger.ErrorContext(ctx, "Não foi possível escutar os eventos da pasta", "target-dir", targetDir)
		return
	}

	logger.InfoContext(ctx, "Iniciando rotina de verificação: ",
		"target-dir", targetDir,
		"images-dir", defaultImagesDir,
		"audios-dir", defaultAudioDir,
		"videos-dir", defaultVideoDir,
		"debug", debug)
	run_watch(ctx, logger, watcher)
}

func run_watch(ctx context.Context, logger *slog.Logger, watcher *fsnotify.Watcher) {
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				fileName := event.Name
				slog.Debug("Novo arquivo criado em", "file-name", fileName)

				// we do that in case of there is any another "." on the file path like:
				// "/path/to/video.file.map"
				lastSuffixIdx := strings.LastIndex(fileName, ".")

				// no prefix with dot found
				if lastSuffixIdx == -1 {
					// TODO: use linux utility to find the type of the file
					// `> file <file_name>`
					logger.InfoContext(ctx, "Sufixo do tipo de arquivo não reconhecido", "file-name", fileName)
					continue
				}

				suffix := fileName[lastSuffixIdx+1:]
				folderTargetName, ok := suffixDirMapping[suffix]
				if !ok {
					logger.InfoContext(ctx, "Sufixo do tipo de arquivo não reconhecido", "file-name", fileName)
					continue
				}

				destPath := filepath.Join(targetDir, folderTargetName, filepath.Base(fileName))
				time.Sleep(2 * time.Second)
				os.Rename(fileName, destPath)
				beeep.Notify("Organizador Go", fmt.Sprintf("Arquivos na pasta [%s => %s] organizados! 📸", fileName, destPath), "")
			}
		case evt := <-watcher.Errors:
			logger.ErrorContext(ctx, "Evento de cancelamento recebido", "error", evt.Error())
			return
		case <-ctx.Done():
			logger.InfoContext(ctx, "Evento de cancelamento do context")
			return
		}
	}
}
