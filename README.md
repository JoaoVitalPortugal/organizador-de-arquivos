# 🚀 Organizador de Downloads (Gopher Edition)

Chega de bagunça na pasta de Downloads! Esse é um projeto em **Go** focado em quem usa **Linux** e quer as coisas nos seus devidos lugares sem esforço. Ele fica vigiando a pasta de Downloads em tempo real e, assim que um arquivo cai lá, o bicho já joga pra pasta certa.

## ✨ O que ele faz?
O programa identifica o arquivo, espera o download terminar (pra não mover arquivo vazio!) e organiza tudo:
- **📸 Imagens:** `.jpg`, `.jpeg`, `.png` ➡ `~/Imagens`
- **🎬 Vídeos:** `.mp4`, `.mkv`, `.mov` ➡ `~/Vídeos`
- **🎧 Áudios:** `.mp3`, `.wav`, `.aac` ➡ `~/Músicas`
- **🔔 Notificação:** Ele te avisa no sistema (pop-up) quando a mágica acontece.

## 🛠️ O que eu usei:
- **Go (Golang):** A linguagem do poder.
- **fsnotify:** Pra ouvir os eventos do Kernel do Linux.
- **beeep:** Pra mandar as notificações bonitinhas pro desktop.

## 🚀 Como rodar essa belezura:

1. **Instale as dependências:**
   ```bash
   go get [github.com/fsnotify/fsnotify](https://github.com/fsnotify/fsnotify)
   go get [github.com/gen2brain/beeep](https://github.com/gen2brain/beeep)# 🚀 Organizador de Downloads (Gopher Edition)


   go get [github.com/fsnotify/fsnotify](https://github.com/fsnotify/fsnotify)
   go get [github.com/gen2brain/beeep](https://github.com/gen2brain/beeep)

2. **Rode o programa**
   ```bash
   go run main.go
