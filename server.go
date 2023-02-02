package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type UserInfo struct {
	Uid  int    `json:"uid"`
	Name string `json:"name"`
}

type WebSockChan struct {
	lock      sync.Mutex
	ch        chan *UserInfo
	sendCh    chan<- *UserInfo
	receiveCh <-chan *UserInfo
}

type WebSockChanStore struct {
	lock sync.Mutex
	m    map[int]*WebSockChan
}

var (
	counter   int
	upgrader  = websocket.Upgrader{}
	wschStore WebSockChanStore
)

func init() {
	wschStore.m = make(map[int]*WebSockChan)
}

func makeWebSockChan() *WebSockChan {
	ch := make(chan *UserInfo)
	return &WebSockChan{lock: sync.Mutex{}, ch: ch, sendCh: ch, receiveCh: ch}
}
func closeWebSockChan(ch *WebSockChan) {
	ch.lock.Lock()
	defer ch.lock.Unlock()
	ch.sendCh = nil
	close(ch.ch)
}
func registerWebSockChan(id int, wch *WebSockChan) {
	wschStore.lock.Lock()
	defer wschStore.lock.Unlock()
	wschStore.m[id] = wch
}
func deregisterWebSockChan(id int) {
	wschStore.lock.Lock()
	defer wschStore.lock.Unlock()
	delete(wschStore.m, id)
}
func iterateWebSockChan(f func(wch *WebSockChan)) {
	wschStore.lock.Lock()
	defer wschStore.lock.Unlock()
	for _, wch := range wschStore.m {
		go f(wch)
	}
}

func sendToWebSock(info *UserInfo) {
	iterateWebSockChan(func(ch *WebSockChan) {
		ch.lock.Lock()
		defer ch.lock.Unlock()
		// lockを取って送信する間に切断が発生した場合、
		// close側が待ちになって送信がブロックしてしまう。
		// 送信は50msで諦める。
		select {
		case ch.sendCh <- info:
		case <-time.After(50 * time.Millisecond):
		}
	})
}

func handleWebSocket(c echo.Context) error {
	c.Logger().Info("handleWebSocket >>>")
	defer c.Logger().Info("handleWebSocket <<<")
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	wch := makeWebSockChan()
	defer closeWebSockChan(wch)
	counter++
	id := counter
	registerWebSockChan(id, wch)
	defer deregisterWebSockChan(id)

	disconnected := make(chan bool)
	defer close(disconnected)

	var wg sync.WaitGroup
	wg.Add(2)
	// 受信
	go func() {
		defer wg.Done()
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				c.Logger().Error(err)
				disconnected <- true
				break
			}
		}
	}()
	// 送信
	go func() {
		defer wg.Done()
	loop:
		for {
			select {
			case info := <-wch.receiveCh:
				err = ws.WriteJSON(*info)
				if err != nil {
					c.Logger().Error(err)
					break loop
				}
			case <-disconnected:
				break loop
			}
		}
	}()
	wg.Wait()
	return nil
}

func handleAuth(c echo.Context) error {
	u := &UserInfo{}
	if err := c.Bind(u); err != nil {
		return err
	}
	sendToWebSock(u)
	return nil
}
func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Static("/", "./app/build")
	e.GET("/ws", handleWebSocket)
	e.POST("/auth", handleAuth)
	e.Logger.SetLevel(log.INFO)
	e.Logger.Fatal(e.Start(":1323"))
}
