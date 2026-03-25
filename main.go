package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/creack/pty"
)

type CommandRequest struct {
	Command string `json:"command"`
}

type CommandResponse struct {
	Output string `json:"output"`
}

type MsfWrapper struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	mutex      sync.Mutex
	lastOutput bytes.Buffer
	running    bool
}

func NewMsfWrapper() (*MsfWrapper, error) {
	cmd := exec.Command("msfconsole", "-q")
	log.Println("[INFO] Starting msfconsole interactively...")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("[ERROR] Failed to start msfconsole: %v\n", err)
		return nil, err
	}
	log.Println("[INFO] msfconsole started (interactive mode, will not exit)")
	mw := &MsfWrapper{cmd: cmd, stdin: ptmx, stdout: ptmx, running: true}
	go mw.readOutputLoop()
	return mw, nil
}

// Continuously read msfconsole output and append to lastOutput
func (m *MsfWrapper) readOutputLoop() {
	reader := bufio.NewReader(m.stdout)
	for m.running {
		line, err := reader.ReadString('\n')
		m.mutex.Lock()
		m.lastOutput.WriteString(line)
		m.mutex.Unlock()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("[ERROR] Error reading msfconsole output: %v\n", err)
			break
		}
	}
}

func (m *MsfWrapper) SendCommand(cmd string) (string, error) {
	log.Printf("[INFO] Sending command to msfconsole: %q\n", cmd)
	_, err := io.WriteString(m.stdin, cmd+"\n")
	if err != nil {
		log.Printf("[ERROR] Failed to write command: %v\n", err)
		return "", err
	}
	return "OK", nil
}

// ObserveLines returns the last n lines of output captured from msfconsole
func (m *MsfWrapper) ObserveLines(n int) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	lines := bytes.Split(m.lastOutput.Bytes(), []byte{'\n'})
	if n > len(lines) {
		n = len(lines)
	}
	if n <= 0 {
		n = 1
	}
	return string(bytes.Join(lines[len(lines)-n:], []byte{'\n'}))
}

func main() {
	// curl -X POST -H "Content-Type: application/json" -d '{"command": "help"}' http://localhost:8080/command
	// curl -X POST -H "Content-Type: application/json"' http://localhost:8080/observe

	msf, err := NewMsfWrapper()
	if err != nil {
		log.Fatalf("[FATAL] Failed to start msfconsole: %v", err)
	}

	// HTTP handler
	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		var req CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		output, err := msf.SendCommand(req.Command)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(CommandResponse{Output: output})
	})

	// Observe handler
	http.HandleFunc("/observe", func(w http.ResponseWriter, r *http.Request) {
		n := 20
		if linesParam := r.URL.Query().Get("lines"); linesParam != "" {
			if parsed, err := strconv.Atoi(linesParam); err == nil && parsed > 0 {
				n = parsed
			}
		}
		output := msf.ObserveLines(n)
		json.NewEncoder(w).Encode(CommandResponse{Output: output})
	})

	// CLI goroutine
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("msf> ")
		for scanner.Scan() {
			cmd := scanner.Text()
			if strings.HasPrefix(cmd, "observe") {
				// Usage: observe -l <n>
				parts := strings.Fields(cmd)
				n := 20
				for i := 1; i < len(parts)-1; i++ {
					if parts[i] == "-l" {
						if parsed, err := strconv.Atoi(parts[i+1]); err == nil && parsed > 0 {
							n = parsed
						}
					}
				}
				fmt.Print(msf.ObserveLines(n))
			} else {
				output, err := msf.SendCommand(cmd)
				if err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Print(output)
				}
			}
			fmt.Print("msf> ")
		}
	}()

	log.Println("[INFO] HTTP API running on :8080 (/command)")
	log.Println("[INFO] msfconsole is running interactively in the background and will not exit until you stop this program.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
