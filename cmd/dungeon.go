package main

import (
	"bytes"
	"dungeon/internal/game"
	"dungeon/internal/lobby"
	"dungeon/internal/transport"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
)

var addr = flag.String("addr", "127.0.0.1:9001", "http service address")
var serveFiles = flag.Bool("serveFiles", true, "use this app to serve static files (js, css, images)")
var appEnv = flag.String("env", "local", "application environment: local, production")

var indexPageContent []byte

func serveIndexPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	_, err := w.Write(indexPageContent)
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/favicon.ico")
}

func main() {
	flag.Parse()

	indexPageContentRaw, err := os.ReadFile("public/index.html")
	if err != nil {
		log.Fatal("Read index.html error: ", err)
	}
	version, err := os.ReadFile("version")
	if err != nil {
		log.Println("Cannot read file 'version': ", err)
	}
	indexPageContent = bytes.Replace(indexPageContentRaw, []byte("%APP_ENV%"), []byte(*appEnv), 1)
	indexPageContent = bytes.Replace(indexPageContent, []byte("%APP_VERSION%"), bytes.TrimSpace([]byte(version)), 2)

	gameMap, err := game.LoadMap("./public/assets/dungeon1.tmj")
	if err != nil {
		log.Fatal("Load map error: ", err)
	}

	// write map to debug light rects
	mapJson, err := json.Marshal(gameMap)
	if err != nil {
		log.Fatal("Marshal map error: ", err)
	}
	err = os.WriteFile("./public/assets/dungeon1_.tmj", mapJson, 0666)
	if err != nil {
		log.Fatal("Write map error: ", err)
	}

	newGameFunc := func(playersClients []lobby.ClientPlayer, room *lobby.Room, broadcastEventFunc func(event interface{})) lobby.GameEventsDispatcher {
		return game.NewGame(playersClients, room, broadcastEventFunc, gameMap)
	}

	newBotFunc := func(botId uint64, room *lobby.Room, sendGameCommand func(client lobby.ClientPlayer, commandName string, commandData json.RawMessage)) lobby.ClientPlayer {
		return game.NewBotClient(botId, room, sendGameCommand)
	}

	matchMaker := game.NewMatchMaker()

	lobbyInstance := lobby.NewLobby(newGameFunc, newBotFunc, matchMaker, 1, 20)
	go lobbyInstance.Run()
	http.HandleFunc("/", serveIndexPage)
	if *serveFiles {
		http.HandleFunc("/favicon.ico", faviconHandler)
		http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./public/js"))))
		http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./public/css"))))
		http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./public/assets"))))
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		transport.ServeWebSocketRequest(lobbyInstance, w, r)
	})
	log.Printf("Listening http://%s", *addr)
	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
