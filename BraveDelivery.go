package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
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
	rows, err := db.Query("SELECT url FROM urls WHERE url LIKE 'https://example.com%'")
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
		b64chunkNumStr, _ := url.QueryUnescape(q.Get("chunk"))
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
	if len(os.Args) < 4 {
		fmt.Println("Usage:")
		fmt.Println("  Encode: program encode <dbfile> <file>")
		fmt.Println("  Decode: program decode <dbfile> <output_dir>")
		return
	}

	mode := os.Args[1]
	dbFile := os.Args[2]

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	switch mode {
	case "encode":
		filePath := os.Args[3]
		encodeFile(filePath, db)

	case "decode":
		outputDir := os.Args[3]
		decodeFile(db, outputDir)

	default:
		fmt.Println("Unknown mode. Use 'encode' or 'decode'.")
	}
}
