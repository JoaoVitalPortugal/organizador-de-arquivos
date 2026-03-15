package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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

	defaultImagesDir = filepath.Join(os.Getenv("HOME"), "Images")
	defaultVideoDir  = filepath.Join(os.Getenv("HOME"), "Vídeos")
	defaultAudioDir  = filepath.Join(os.Getenv("HOME"), "Músicas")

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

// Function will receive "baseTargetPath" which is the directory where we create new folders
// which will have each type of media file. Target folder will be the name of the folder for given media.
func moveFileToPath(baseTargetPath, targetFolder, currentFilePath string) error {
	absFilePath, err := filepath.Abs(filepath.Join(baseTargetPath, currentFilePath))
	if err != nil {
		return fmt.Errorf("Não foi possível resolver o path absoluto do arquivo [%s] :%w", currentFilePath, err)
	}

	absTargetFolder, err := filepath.Abs(targetFolder)
	if err != nil {
		return fmt.Errorf("Não foi possível resolver o path absoluto da pasta [%s] :%w", targetFolder, err)
	}

	_, err = os.Stat(absTargetFolder)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(absTargetFolder, os.FileMode(0755))
		if err != nil {
			return fmt.Errorf("Não foi possível criar a pasta de destino [%s]: %w", currentFilePath, err)
		}
	}

	slog.Info("Movendo arquivo para novo local",
		"detected-file-path", absFilePath,
		"target-file-path", filepath.Join(absTargetFolder, currentFilePath))
	os.Rename(absFilePath, filepath.Join(absTargetFolder, currentFilePath))
	return nil
}

// only work on linux systems
func isFileInUse(filePath string) bool {
	return exec.Command("fuser", filePath).Run() == nil
}

func waitUntilFileIsFree(ctx context.Context, logger *slog.Logger, filePath string, interval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !isFileInUse(filePath) {
				return nil
			}
			logger.Debug("Arquivo ainda está em uso, aguardando...", "file", filepath.Base(filePath))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
			}
		}
	}
}

func run_watch(ctx context.Context, logger *slog.Logger, watcher *fsnotify.Watcher) {
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				fileName := filepath.Base(event.Name)
				logger.DebugContext(ctx, "Novo arquivo criado em", "file-name", fileName)

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
					logger.WarnContext(ctx, "Sufixo do tipo de arquivo não reconhecido", "file-name", fileName)
					continue
				}

				if err := waitUntilFileIsFree(ctx, logger, event.Name, 1*time.Second); err != nil {
					logger.WarnContext(ctx, "Cancelado enquanto aguardava arquivo", "file-name", fileName, "error", err)
					continue
				}

				if err := moveFileToPath(defaultDir, folderTargetName, fileName); err != nil {
					logger.ErrorContext(ctx, "Não conseguimos mover o arquivo para pasta", "error", err)
					continue
				}

				beeep.Notify("Organizador Go",
					fmt.Sprintf("Arquivos na pasta [%s => %s] organizados! 📸", fileName, folderTargetName),
					"")
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
