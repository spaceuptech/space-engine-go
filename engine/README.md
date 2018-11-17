# engine
--
    import "spaceuptech.com/space-engine-go/engine"


## Usage

```go
// Function to be registered
func myFunc(params engine.M, auth engine.M, cb engine.CallBack) {
    log.Println("Params", params, "Auth", auth)
    // Do something

    // Call the callback
    cb(engine.TypeResponse, engine.M{"ack": true})
}

// Create an instance of engine
myEngine, err := engine.Init("my-engine", "")
if err != nil {
    log.Println("Err", err)
    return
}

// Register function
myEngine.RegisterFunc("my-func", myFunc)

// Start engine
myEngine.Start()

// Call function of some other engine
res, err := myEngine.Call("some-engine", "some-func", engine.M{"msg": "space-engine-go is awesome!"}, 1000)
log.println("Res", res, "Err", err)

```