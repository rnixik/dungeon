package main

import (
	"bytes"
	"dungeon/internal/game"
	"dungeon/internal/lobby"
	"dungeon/internal/transport"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var addr = flag.String("addr", "127.0.0.1:9001", "http service address")
var serveFiles = flag.Bool("serveFiles", true, "use this app to serve static files (js, css, images)")
var appEnv = flag.String("env", "local", "application environment: local, production")
var mapPath = flag.String("map", "./public/assets/dungeon1.tmj", "path to the .tmj map (or room-template map when --rooms > 0) to load")
var numRooms = flag.Int("rooms", 10, "number of rooms to assemble from the template's predefined rooms; 0 loads the map as-is")
var mapSeed = flag.Int64("seed", 0, "random seed for map generation; 0 derives one from the current time")

var indexPageContent []byte

type avatarCacheEntry struct {
	data        []byte
	contentType string
}

var avatarCache sync.Map // map[string]*avatarCacheEntry

var avatarHTTPClient = &http.Client{Timeout: 10 * time.Second}

func avatarProxyHandler(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	host := parsed.Hostname()
	if host != "t.me" {
		if *appEnv != "local" {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
	}

	if cached, ok := avatarCache.Load(rawURL); ok {
		entry := cached.(*avatarCacheEntry)
		w.Header().Set("Content-Type", entry.contentType)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(entry.data)
		return
	}

	resp, err := avatarHTTPClient.Get(rawURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "read failed", http.StatusBadGateway)
		return
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}

	avatarCache.Store(rawURL, &avatarCacheEntry{data: data, contentType: ct})

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(data)
}

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

	var gameMap *game.Map
	if *numRooms > 0 {
		gameMap, err = game.LoadGeneratedMap(*mapPath, *numRooms, *mapSeed)
	} else {
		gameMap, err = game.LoadMap(*mapPath)
	}
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
		return game.NewGame(playersClients, room, broadcastEventFunc, gameMap, *appEnv == "local")
	}

	newBotFunc := func(botId uint64, room *lobby.Room, sendGameCommand func(client lobby.ClientPlayer, commandName string, commandData json.RawMessage)) lobby.ClientPlayer {
		return game.NewBotClient(botId, room, sendGameCommand)
	}

	matchMaker := game.NewMatchMaker()

	lobbyInstance := lobby.NewLobby(newGameFunc, newBotFunc, matchMaker, 1, 20)
	go lobbyInstance.Run()
	http.HandleFunc("/", serveIndexPage)
	http.HandleFunc("/avatar-proxy", avatarProxyHandler)
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
