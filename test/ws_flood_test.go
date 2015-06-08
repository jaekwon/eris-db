package test

import (
	"github.com/eris-ltd/erisdb/server"
	"github.com/eris-ltd/erisdb/test/client"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const CONNS = 100
const MESSAGES = 1000

// To keep track of new websocket sessions on the server.
type SessionCounter struct {
	opened int
	closed int
}

func (this *SessionCounter) Run(oChan, cChan <-chan *server.WSSession) {
	go func() {
		for {
			select {
			case <-oChan:
				this.opened++
				break
			case <-cChan:
				this.closed++
				break
			}
		}
	}()
}

func (this *SessionCounter) Report() (int, int, int) {
	return this.opened, this.closed, this.opened - this.closed
}

// Coarse flood testing just to ensure that websocket server
// does not crash.
func TestWsFlooding(t *testing.T) {

	// New websocket server.
	wsServer := NewScumsocketServer(CONNS)
	
	// Keep track of sessions.
	sc := &SessionCounter{}
	
	// Register the observer.
	oChan := wsServer.SessionManager().SessionOpenEventChannel()
	cChan := wsServer.SessionManager().SessionCloseEventChannel()

	sc.Run(oChan, cChan)

	serveProcess := NewServeScumSocket(wsServer)
	errServe := serveProcess.Start()
	assert.NoError(t, errServe, "ScumSocketed!")
	t.Logf("Flooding...")
	// Run
	errRun := runWs()

	errStop := serveProcess.Stop(time.Millisecond * 100)
	assert.NoError(t, errRun, "ScumSocketed!")
	assert.NoError(t, errStop, "ScumSocketed!")
	o, c, a := sc.Report() 
	assert.Equal(t, o, CONNS, "Server registered '%d' opened conns out of '%d'", o, CONNS)
	assert.Equal(t, c, CONNS, "Server registered '%d' closed conns out of '%d'", c, CONNS)
	assert.Equal(t, a, 0, "Server registered '%d' conns still active after closing all.", a)

	t.Logf("WebSocket test: A total of %d messages sent succesfully over %d parallel websocket connections.\n", CONNS*MESSAGES, CONNS)
}

func runWs() error {
	doneChan := make(chan bool)
	errChan := make(chan error)
	for i := 0; i < CONNS; i++ {
		go wsClient(doneChan, errChan)
	}
	runners := 0
	for runners < CONNS {
		select {
		case _ = <-doneChan:
			runners++
		case err := <-errChan:
			return err
		}
	}
	return nil
}

func wsClient(doneChan chan bool, errChan chan error) {
	client := client.NewWSClient("ws://localhost:31337/scumsocket")
	_, err := client.Dial()
	if err != nil {
		errChan <- err
		return
	}
	readChan := client.Read()
	i := 0
	for i < MESSAGES {
		client.WriteMsg([]byte("test"))
		<-readChan
		i++
	}

	client.Close()
	time.Sleep(time.Millisecond * 500)
	doneChan <- true
}