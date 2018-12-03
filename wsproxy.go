// stratum proxy
// allows a browser to connect over websockets and passes
// the message to the mining pool over tcp
// then passes the response back to the browser over websockets
package main

import (
	"bufio"
	"fmt"
	"os"
	// "encoding/json"
	// "fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

var poolHost string
var poolPort string
var poolAddr string

var up = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
	Subprotocols:    []string{"binary"},
}

/*
 methods:
	login(login)
	getjob()
	submit(job_id, nonce)
	keepalived()
*/

// StratumMessage are JSON-RPC messages
type StratumMessage struct {
	ID     int                    `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

func setFromEnv(name string, missing string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return missing
}

func init() {
	poolHost = setFromEnv("POOL_HOST", "localhost")
	poolPort = setFromEnv("POOL_PORT", "4444")
	poolAddr = fmt.Sprintf("%s:%s", poolHost, poolPort)
}

// handle a miner connection by creating a proxy to the pool
func handler(w http.ResponseWriter, r *http.Request) {
	// log.Printf("req: %v", r)
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("wsproxy: handler: " + err.Error())
		return
	}

	log.Printf("miner connected: %s", conn.RemoteAddr())

	// make a connection to the pool for this miner...
	pool, err := net.Dial("tcp", poolAddr)
	if err != nil {
		log.Printf("wsproxy: failed to connect to pool: " + err.Error())
		return
	}
	log.Println("connected to pool.")

	go func() {
		for {
			// copy from miner to pool
			_, rdr, err := conn.NextReader()
			if err != nil {
				log.Printf("disconnect: %v\n", err)
				pool.Close()
				return
			}

			log.Println("miner --> pool")

			w := bufio.NewWriter(pool)
			if _, err := io.Copy(w, rdr); err != nil {
				return
			}
		}
	}()

	go func() {
		for {
			// copy from pool to miner
			line, err := bufio.NewReader(pool).ReadString('\n')
			if err != nil {
				log.Printf("disconnect: %v\n", err)
				conn.Close()
				return
			}
			log.Println("pool --> miner")
			conn.WriteMessage(websocket.BinaryMessage, []byte(line))
			/*
				rdr := bufio.NewReader(pool)
				log.Println("pool --> miner")
				w, err := conn.NextWriter(websocket.BinaryMessage)

				if err != nil {
					panic("failed to create writer!")
				}
				if _, err := io.Copy(w, rdr); err != nil {
					return
				}
				if err := w.Close(); err != nil {
					return
				}
			*/
		}
	}()
}

func main() {
	log.Printf("wsproxy pool=%s", poolAddr)
	http.HandleFunc("/", handler)

	if err := http.ListenAndServe("0.0.0.0:8888", nil); err != nil {
		panic("ListenAndServer: " + err.Error())
	}
}
