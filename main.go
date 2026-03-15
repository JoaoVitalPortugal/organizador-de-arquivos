package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify" // Caminho correto sem o ://
	"github.com/gen2brain/beeep"
)

func main() {
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()

	path := filepath.Join(os.Getenv("HOME"), "Downloads")
	watcher.Add(path)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				nomeArquivo := event.Name
				home, _ := os.UserHomeDir()
				// ESSE É PARA AS IMAGENS
				if strings.HasSuffix(nomeArquivo, ".jpeg") || strings.HasSuffix(nomeArquivo, ".jpg") || strings.HasSuffix(nomeArquivo, ".png") {
					// MUDE O "IMAGENS" PARA O NOME DA PASTA DESEJADA QUE FIQUE AS IMAGENS, UM EXEMPLO: PICTURES
					destino := filepath.Join(home, "Imagens", filepath.Base(nomeArquivo))
					time.Sleep(2 * time.Second)
					os.Rename(nomeArquivo, destino)
					beeep.Notify("Organizador Go", "Arquivos organizado! 📸", "")
					// ESSE É PARA OS VIDEOS
				} else if strings.HasSuffix(nomeArquivo, ".mp4") || strings.HasSuffix(nomeArquivo, ".mkv") || strings.HasSuffix(nomeArquivo, ".mov") {
					// Troque "Vídeos" pelo nome da sua pasta de videos
					destino := filepath.Join(home, "Vídeos", filepath.Base(nomeArquivo))
					time.Sleep(2 * time.Second)
					os.Rename(nomeArquivo, destino)
					beeep.Notify("Organizador Go", "Arquivos organizado! 📸", "")
					// Esse é para musicas/audios
				} else if strings.HasSuffix(nomeArquivo, ".mp3") || strings.HasSuffix(nomeArquivo, ".wav") || strings.HasSuffix(nomeArquivo, ".aac") {
					// troque "músicas" pelo nome da sua pasta de musicas/audios
					destino := filepath.Join(home, "Músicas", filepath.Base(nomeArquivo))
					time.Sleep(2 * time.Second)
					os.Rename(nomeArquivo, destino)
					beeep.Notify("Organizador Go", "Arquivos organizado! 📸", "")
				}
			}
		case <-watcher.Errors:
			return
		}
	}
}
