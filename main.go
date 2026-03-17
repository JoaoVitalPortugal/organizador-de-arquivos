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
	debug     bool

	defaultTargetDir = filepath.Join(os.Getenv("HOME"), "Downloads")
	defaultDestDir = os.Getenv("HOME")

	defaultImagesDir = filepath.Join(os.Getenv("HOME"), "Images")
	defaultVideoDir  = filepath.Join(os.Getenv("HOME"), "Vídeos")
	defaultAudioDir  = filepath.Join(os.Getenv("HOME"), "Músicas")


	// Essa mapa vai ser usado para guardar um mapeamento entre o tipo de arquivo
	// e a pasta para onde ele deve ser movido.
	//
	// E.g.:
	//
	//
	// fileType := "jpeg"
	// 
	// targetFolder, ok := suffixDirMapping[fileType]
	//
	// if !ok { "verifica se o tipo de arquivo não é suportado e então não faz nada" }
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
	// Usando o pacote "flag" builtin do Go, podemos receber como variáveis 
	// os argumentos passados na linha de comando.
	// Como por exemplo, se o usuário executar o programa assim: 
	// 
	// > go build . -o organizador && ./organizador --dest-dir "/home/<user>/my-media-files"
	// 
	// Usaremos o "dest-dir" como diretório raiz para todas as pastas de mídia. Ficando:
	//  
	//	/home/<user>/my-media-files
	//  ├── Videos (valor padrão)
	//  └── Músicas (valor padrão)
	//
	// Assim como se o usuário iniciar o programa como:
	//
	// > go build . -o organizador && ./organizador --images-dir "MinhasImagens"
	//
	// Usaremos o "dest-dir" como o valor padrão "$HOME" e o "images-dir" para o nome da pasta de imagens, ficando:
	//
	//	/home/<user>
	//  ├── MinhasImagens
	//  └── Músicas (valor padrão)
	flag.StringVar(&defaultTargetDir, "target-dir", defaultTargetDir, `Diretório de onde o app vai escutar novos arquivos criados.`)
	flag.StringVar(&defaultDestDir, "dest-dir", defaultTargetDir, `Diretório para onde o app vai criar as pastas para organizar.`)
	flag.StringVar(&defaultImagesDir, "images-dir", defaultImagesDir, `Diretório para onde movemos arquivos de imagem.`)
	flag.StringVar(&defaultAudioDir, "audios-dir", defaultAudioDir, `Diretório para onde movemos arquivos de áudio.`)
	flag.StringVar(&defaultVideoDir, "videos-dir", defaultVideoDir, `Diretório para onde movemos arquivos de vídeo.`)
	flag.BoolVar(&debug, "debug", false, `Habilita log de debug. Por padrão é desabilitado`)
	flag.Parse()

	// Usamos a biblioteca padrão do Go, "slog", para gerenciar os logs da aplicação.
	// Aqui também escolhemos entre o modo de debug or normal (info).
	// No modo debug vão aparecer logs que ajudam a investigar erros mas não úties ao usuário normal.
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
		logger.ErrorContext(ctx, "não foi possível criar um 'watcher' para o path", "target-dir", defaultTargetDir)
	}
	defer watcher.Close()

	err = watcher.Add(defaultTargetDir)
	if err != nil {
		logger.ErrorContext(ctx, "Não foi possível escutar os eventos da pasta", "target-dir", defaultTargetDir)
		return
	}

	logger.InfoContext(ctx, "Iniciando rotina de verificação: ",
		"target-dir", defaultTargetDir,
		"dest-dir", defaultDestDir,
		"images-dir", defaultImagesDir,
		"audios-dir", defaultAudioDir,
		"videos-dir", defaultVideoDir,
		"debug", debug)

	run_watch(ctx, logger, watcher)
}

// Função verifica os diretórios usados para organizar as mídias e cria-los se for preciso:
//
// "baseTargetPath": é o diretório raiz onde escutamos novos arquivos criados.
//
// "targetFolder": o diretário onde devemos usar como base para criar os diretórios de organização se não existirem
//
// "fileName": nome do arquivo que deve ser movido.
func moveFileToPath(baseTargetPath, targetFolder, fileName string) error {
	absFilePath, err := filepath.Abs(filepath.Join(baseTargetPath, fileName))
	if err != nil {
		return fmt.Errorf("Não foi possível resolver o path absoluto do arquivo [%s] :%w", fileName, err)
	}

	absTargetFolder, err := filepath.Abs(filepath.Join(defaultDestDir, targetFolder))
	if err != nil {
		return fmt.Errorf("Não foi possível resolver o path absoluto da pasta [%s] :%w", targetFolder, err)
	}

	_, err = os.Stat(absTargetFolder)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(absTargetFolder, os.FileMode(0755))
		if err != nil {
			return fmt.Errorf("Não foi possível criar a pasta de destino [%s]: %w", fileName, err)
		}
	}

	slog.Info("Movendo arquivo para novo local",
		"detected-file-path", absFilePath,
		"target-file-path", filepath.Join(absTargetFolder, fileName))
	os.Rename(absFilePath, filepath.Join(absTargetFolder, fileName))
	return nil
}

// Somente funciona no linux. Chama o cli "fuser" para verificar se o arquivo ainda tem file-descriptor ativos.
func isFileInUse(filePath string) bool {
	return exec.Command("fuser", filePath).Run() == nil
}

// Loop que usa "interval" para a cada invervale testar novamente se o arquivo continua sendo escrito.
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

				// Pegamos o sufixo do tipo de arquivo assim em caso de ter algum path
				// possa ter outras ocorrência do delimitador "." como:
				// "/path/to/video.file.map"
				lastSuffixIdx := strings.LastIndex(fileName, ".")

				// Não foi achado nenhum sufixo no arquivo.
				if lastSuffixIdx == -1 {
					// TODO: usar uma ferramenta linux ou uma biblioteca para verificar 
					// o tipo do arquivo sme precisar da extensão.
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

				if err := moveFileToPath(defaultTargetDir, folderTargetName, fileName); err != nil {
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
