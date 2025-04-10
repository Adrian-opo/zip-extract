package main

import (
	"bufio"
	"fmt"
	"github.com/alexmullins/zip"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Global WebSocket connection manager
var (
	wsConnections = make(map[*websocket.Conn]bool)
	wsLock        sync.Mutex
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

// Message struct for WebSocket communications
type Message struct {
	Type         string   `json:"type"`
	Password     string   `json:"password,omitempty"`
	Status       string   `json:"status,omitempty"`
	Progress     int      `json:"progress,omitempty"`
	Total        int      `json:"total,omitempty"`
	RecentFailed []string `json:"recentFailed,omitempty"`
	Success      bool     `json:"success,omitempty"`
}

func broadcastMessage(msg Message) {
	wsLock.Lock()
	defer wsLock.Unlock()
	
	for conn := range wsConnections {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("WebSocket write error: %v", err)
			conn.Close()
			delete(wsConnections, conn)
		}
	}
}

func main() {
	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	})
	r.LoadHTMLGlob("templates/*")
	r.Static("/uploads", "./uploads")
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"status":        "",
			"failedAttempts": []string{},
			"pagination": gin.H{
				"currentPage": 1,
				"totalPages":  1,
				"pageSize":    10,
			},
		})
	})

	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if (err != nil) {
			log.Printf("Failed to upgrade WebSocket connection: %v", err)
			return
		}
		
		wsLock.Lock()
		wsConnections[conn] = true
		wsLock.Unlock()
		
		defer func() {
			wsLock.Lock()
			delete(wsConnections, conn)
			wsLock.Unlock()
			conn.Close()
		}()
		
		// Simple ping-pong to keep the connection alive
		for {
			messageType, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				break
			}
			
			if messageType == websocket.PingMessage {
				if err := conn.WriteMessage(websocket.PongMessage, []byte{}); err != nil {
					log.Printf("WebSocket write error: %v", err)
					break
				}
			}
		}
	})

	r.POST("/upload", func(c *gin.Context) {
		zipFile, err := c.FormFile("zipfile")
		if err != nil {
			c.HTML(400, "index.html", gin.H{
				"status":        "Erro ao enviar arquivo ZIP",
				"failedAttempts": []string{},
			})
			return
		}

		wordlistFile, err := c.FormFile("wordlist")
		if err != nil {
			c.HTML(400, "index.html", gin.H{
				"status":        "Erro ao enviar wordlist",
				"failedAttempts": []string{},
			})
			return
		}

		zipPath := filepath.Join("uploads", zipFile.Filename)
		wordlistPath := filepath.Join("uploads", wordlistFile.Filename)

		// Create uploads directory if it doesn't exist
		if _, err := os.Stat("uploads"); os.IsNotExist(err) {
			os.Mkdir("uploads", 0755)
		}

		c.SaveUploadedFile(zipFile, zipPath)
		c.SaveUploadedFile(wordlistFile, wordlistPath)
		
		// Return immediate response to show loading UI
		c.HTML(200, "index.html", gin.H{
			"status":         "Iniciando processo de quebra de senha...",
			"loading":        true,
			"zipFilename":    zipFile.Filename,
			"wordlistName":   wordlistFile.Filename,
		})
		
		// Start password cracking process in a goroutine
		go func() {
			password, found, failedAttempts, totalWords := crackZipWithProgress(zipPath, wordlistPath)
			
			// Pagination setup
			pageSize := 10
			totalPages := (len(failedAttempts) + pageSize - 1) / pageSize
			if totalPages == 0 {
				totalPages = 1
			}
			
			// Broadcast final result
			if found {
				broadcastMessage(Message{
					Type:     "complete",
					Password: password,
					Status:   fmt.Sprintf("Senha encontrada: %s", password),
					Success:  true,
					Progress: totalWords,
					Total:    totalWords,
				})
			} else {
				broadcastMessage(Message{
					Type:     "complete",
					Status:   "Nenhuma senha funcionou",
					Success:  false,
					Progress: totalWords,
					Total:    totalWords,
				})
			}
		}()
	})

	r.Run(":8080")
}

func crackZipWithProgress(zipPath, wordlistPath string) (string, bool, []string, int) {
	failedAttempts := []string{}
	recentFailed := []string{}
	totalWords := countWordsInFile(wordlistPath)
	processedWords := 0
	
	// Initial progress update
	broadcastMessage(Message{
		Type:     "progress",
		Status:   "Iniciando...",
		Progress: 0,
		Total:    totalWords,
	})
	
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		broadcastMessage(Message{
			Type:   "error",
			Status: "Erro ao abrir arquivo ZIP",
		})
		return "", false, failedAttempts, totalWords
	}
	defer r.Close()

	file, err := os.Open(wordlistPath)
	if err != nil {
		broadcastMessage(Message{
			Type:   "error",
			Status: "Erro ao abrir wordlist",
		})
		return "", false, failedAttempts, totalWords
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		password := scanner.Text()
		success := false
		processedWords++
		
		// Update progress every 10 attempts or for each attempt if total < 100
		if processedWords % 10 == 0 || totalWords < 100 {
			// Calculate progress percentage (removed the unused variable)
			broadcastMessage(Message{
				Type:         "progress",
				Status:       fmt.Sprintf("Testando: %s", password),
				Progress:     processedWords,
				Total:        totalWords,
				RecentFailed: recentFailed,
			})
		}
		
		for _, f := range r.File {
			f.SetPassword(password)
			rc, err := f.Open()
			if err == nil {
				_, err = io.ReadAll(rc)
				rc.Close()
				if err == nil {
					success = true
					break
				}
			}
		}
		
		if success {
			return password, true, failedAttempts, totalWords
		} else {
			failedAttempts = append(failedAttempts, password)
			
			// Keep only last 10 failed attempts for real-time display
			recentFailed = append(recentFailed, password)
			if len(recentFailed) > 10 {
				recentFailed = recentFailed[1:]
			}
		}
	}
	
	return "", false, failedAttempts, totalWords
}

func countWordsInFile(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}

	return lineCount
}
