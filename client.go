package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var ServPort string
var ServUrl string
var (
	pidChan chan int
	alive   bool
)

func main() {

	fmt.Printf("please input the server addresse:\n")
	_, err := fmt.Scanf("%v", &ServUrl)
	if err != nil {
		fmt.Printf("error in addresse: %v", err)
		return
	}
	fmt.Printf("please input the port(press enter for default :8080):\n")
	_, err = fmt.Scanf("%v", &ServPort)
	if err != nil {
		ServPort = "8080"
		fmt.Printf("dafault port slected\n")
	}
	ServUrl = strings.TrimSpace(ServUrl)
	ServPort = strings.TrimSpace(ServPort)
	url := fmt.Sprintf("ws://%v:%v", ServUrl, ServPort)
	alive = true
	pidChan = make(chan int, 1)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Printf("trying to connect... \n")
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				fmt.Printf("%v", err)
				time.Sleep(time.Second)
			} else if conn != nil {
				go clientStarter(conn, ctx, stop)
				pid := <-pidChan
				p, err := os.FindProcess(pid)
				if err != nil {
					return
				}
				select {
				case <-ctx.Done():
					alive = false
					err := conn.Close()
					if err != nil {
						return
					}
					fmt.Printf("\n\nclosing....")
					err = p.Signal(syscall.SIGTERM)
					if err != nil {
						return
					}

				}
			}
		}
	}
}

func clientStarter(conn *websocket.Conn, ctx context.Context, stop context.CancelFunc) {
	pidChan <- os.Getpid()

	reader := bufio.NewReader(os.Stdin)

	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
		}
	}(conn)
	go Receive(conn, stop)
	fmt.Println()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context canceled, stopping clientStarter.")

			return
		default:
			text, _ := reader.ReadString('\n')
			fmt.Printf("\t|----messageSent---->")
			if alive {
				text = strings.TrimSpace(text)
				if len(text) == 0 {
					continue
				}
				err := conn.WriteMessage(websocket.TextMessage, []byte(text))
				if err != nil {
					fmt.Println("Error connecting")
					break
				}
			}
		}
	}
}

func Receive(conn *websocket.Conn, stop context.CancelFunc) {
	writer := bufio.NewWriter(os.Stdout)
	for {
		_, text, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("connection terminated shutting down")
			stop()
			return
		}
		_, err = writer.WriteString("\n" + string(text) + "\t|------------------->")
		if err != nil {
			continue
		}
		err = writer.Flush()
		if err != nil {
			return
		}
	}
}
