package engine

import (
	"encoding/json"
	"log"
	"time"

	nats "github.com/nats-io/go-nats"
)

const (
	// TypeResponse is used as callback type to give response
	TypeResponse = "response"
)

// M is the type for json objects
type M map[string]interface{}

// CallBack is the callback function to be called by the function
type CallBack func(string, M)

// Func is the function to be registered
type Func func(M, M, CallBack)

// Engine Object
type Engine struct {
	name       string
	funcs      map[string]Func
	natsClient *nats.Conn
	ch         chan *nats.Msg
}

// Init initializes a FaaS engine
func Init(name string, url string) (*Engine, error) {
	if url == "" {
		url = nats.DefaultURL
	}
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	ch := make(chan *nats.Msg, 10)

	engine := &Engine{name: name, funcs: map[string]Func{}, natsClient: nc, ch: ch}
	return engine, nil
}
func (engine *Engine) callFunc(fn Func, params M, auth M, replyTo string) {
	fn(params, auth, func(cbType string, res M) {
		switch cbType {
		case "response":
			b, err := json.Marshal(res)
			if err != nil {
				engine.natsClient.Publish(replyTo, []byte("{\"ack\":false}"))
				return
			}
			engine.natsClient.Publish(replyTo, b)
		}
	})
}

// RegisterFunc registers a function to an engine with the given name and options
func (engine *Engine) RegisterFunc(name string, fn Func) error {

	subj := "faas:" + engine.name + ":" + name

	_, present := engine.funcs[subj]

	if !present {
		_, err := engine.natsClient.ChanQueueSubscribe(subj, engine.name, engine.ch)
		if err != nil {
			return err
		}
		engine.funcs[subj] = fn
	}
	return nil
}

// Start starts the engine
func (engine *Engine) Start() {
	for msg := range engine.ch {
		fn, present := engine.funcs[msg.Subject]
		if present {
			data := msg.Data
			obj := M{}
			err := json.Unmarshal(data, &obj)
			if err != nil {
				log.Println("Err - ", err)
				engine.natsClient.Publish(msg.Reply, []byte("{\"ack\":false}"))
				continue
			}

			paramsTemp, p := obj["params"]
			if !p {
				log.Println("Err - Params not present")
				engine.natsClient.Publish(msg.Reply, []byte("{\"ack\":false}"))
				continue
			}

			params, ok := paramsTemp.(map[string]interface{})
			if !ok {
				log.Println("Err - Params of incorrect type")
				engine.natsClient.Publish(msg.Reply, []byte("{\"ack\":false}"))
				continue
			}

			var auth M
			authTemp, p := obj["auth"]
			if p {
				var ok bool
				auth, ok = authTemp.(map[string]interface{})
				if !ok {
					log.Println("Err - Auth obj of incorrect type")
					engine.natsClient.Publish(msg.Reply, []byte("{\"ack\":false}"))
					continue
				}
				engine.callFunc(fn, params, auth, msg.Reply)
				continue
			}
			engine.callFunc(fn, params, nil, msg.Reply)
		}
	}
}

// Call -- calls a function of any engine
func (engine *Engine) Call(engineName, functionName string, params map[string]interface{}, timeOut int) (*M, error) {
	subj := "faas:" + engineName + ":" + functionName

	// Convert params into json
	dataBytes, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	// Make a nats request
	msg, err := engine.natsClient.Request(subj, dataBytes, time.Duration(timeOut)*time.Millisecond)
	if err != nil {
		return nil, err
	}

	// Parse msg
	res := M{}
	err = json.Unmarshal(msg.Data, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
